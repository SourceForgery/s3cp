package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/amoghe/distillog"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jessevdk/go-flags"
	"io"
	"net/url"
	"os"
	"s3cp/logger"
	"strings"
	"sync"
)

var Commit = "not set"

var Options struct {
	Destination string `short:"d" long:"destination" description:"Setting this makes all the arguments at the end become sources for use with e.g. xargs"`
	Version     bool   `short:"V" long:"version" description:"Print version"`
	Verbose     bool   `short:"v" long:"verbose" description:"Verbose"`
	AccessKey   string `long:"access-key" description:"Access key"`
	SecretKey   string `long:"secret-key" description:"Secret key"`
	WriteKeys   bool   `long:"write-keys" description:"Write access key and secret key to ~/.aws to avoid having it visible in /bin/ps aux"`
}

var bucketToHost = make(map[string]types.BucketLocationConstraint)

var mutex = &sync.RWMutex{}

func resolveBucket(ctx context.Context, client *s3.Client, bucket string) (region types.BucketLocationConstraint, err error) {
	mutex.RLock()
	region = bucketToHost[bucket]
	mutex.RUnlock()
	if region != "" {
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if region = bucketToHost[bucket]; region != "" {
		return
	}
	response, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: aws.String(bucket)})
	if err == nil {
		region = response.LocationConstraint
		bucketToHost[bucket] = region
	}
	return
}

type uploadFunc func(filename string, data io.Reader) (err error)

var log distillog.Logger

func doEverything() (err error) {
	ctx := context.Background()
	args, err := flags.ParseArgs(&Options, os.Args)
	if err != nil {
		return
	}

	var minLogLevel logger.LogLevel
	if Options.Verbose {
		minLogLevel = logger.Debug
	} else {
		minLogLevel = logger.Info
	}
	log = logger.ConfigurableVerboseLogger{
		ProxyLog:    distillog.NewStdoutLogger("s3cp"),
		MinLogLevel: minLogLevel,
	}

	if Options.Version {
		log.Errorf("Compiled from %s", Commit)
		os.Exit(0)
	}

	hasAccessKey := Options.AccessKey != ""
	hasSecretKey := Options.SecretKey != ""
	if hasAccessKey != hasSecretKey {
		return errors.New("either none or both of --access-key and --secret-key")
	}

	if Options.WriteKeys {
		if err = writeConfig(hasAccessKey); err != nil {
			return
		}
		log.Infoln("Wrote config. Exiting")
		os.Exit(0)
	}

	sources := args[1:]
	if len(sources) == 0 {
		return errors.New("not enough arguments")
	}

	var dest string
	if "" == Options.Destination {
		dest = sources[len(sources)-1]
		sources = sources[:len(sources)-1]
	} else {
		dest = Options.Destination
	}

	if len(sources) == 0 {
		return errors.New("not enough arguments")
	}

	cfg, err := config.LoadDefaultConfig(ctx, func(options *config.LoadOptions) error {
		if hasAccessKey {
			options.Credentials = credentials.NewStaticCredentialsProvider(Options.AccessKey, Options.SecretKey, "")
		}
		options.Region = "us-west-2"
		return nil
	})
	if err != nil {
		return
	}
	client := s3.NewFromConfig(cfg)

	if len(sources) > 1 && !strings.HasSuffix(dest, "/") {
		return errors.New(fmt.Sprintf("Must have a trailing slash (directory indicator) when copying multiple files"))
	}

	uploadFn, err := prepareDestinationFunction(dest, ctx, client)
	if err != nil {
		return
	}

	successful := true
	var wg sync.WaitGroup
	for _, source := range sources {
		wg.Add(1)
		go func(source string) {
			defer wg.Done()
			if strings.HasPrefix(source, "s3://") {
				err = copyFromS3(source, ctx, client, uploadFn)
			} else {
				err = copyFromLocalFilesystem(source, uploadFn)
			}
			if err != nil {
				successful = false
				log.Errorf("Failed to copy '%s': %s", source, err)
			} else {
				log.Debugf("Copied '%s' successfully", source)
			}
		}(source)
	}
	wg.Wait()
	if !successful {
		os.Exit(1)
	}
	return
}

func copyFromLocalFilesystem(source string, uploadFn uploadFunc) (err error) {
	file, err := os.Open(source)
	if err != nil {
		return
	}
	defer func(file *os.File) { _ = file.Close() }(file)
	if err = uploadFn(source, file); err != nil {
		return
	}

	err = file.Close()
	return
}

func copyFromS3(source string, ctx context.Context, client *s3.Client, uploadFn uploadFunc) (err error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return
	}

	region, err := resolveBucket(ctx, client, parsed.Hostname())
	if err != nil {
		return
	}

	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(parsed.Hostname()),
		Key:    aws.String(strings.TrimPrefix(parsed.Path, "/")),
	}
	output, err := client.GetObject(
		ctx,
		getObjectInput,
		func(options *s3.Options) {
			options.Region = string(region)
		},
	)
	if err != nil {
		return
	}

	defer func(Body io.ReadCloser) { _ = Body.Close() }(output.Body)

	if err = uploadFn(source, output.Body); err != nil {
		return
	}
	err = output.Body.Close()
	return
}

func prepareDestinationFunction(dest string, ctx context.Context, client *s3.Client) (result uploadFunc, err error) {
	if strings.HasPrefix(dest, "s3://") {
		var parsed *url.URL
		parsed, err = url.Parse(dest)
		if err != nil {
			return
		}
		if parsed.Path == "" {
			return nil, errors.New(fmt.Sprintf("no path in %s", dest))
		}

		result = func(filename string, data io.Reader) (err error) {
			var key string
			if strings.HasSuffix(dest, "/") {
				key = parsed.Path + filename[strings.LastIndex(filename, "/")+1:]
			} else {
				key = parsed.Path
			}

			var region types.BucketLocationConstraint
			region, err = resolveBucket(ctx, client, parsed.Hostname())
			if err != nil {
				return err
			}

			_, err = client.PutObject(
				ctx,
				&s3.PutObjectInput{
					Bucket: aws.String(parsed.Hostname()),
					Key:    aws.String(strings.TrimPrefix(key, "/")),
					Body:   data,
				},
				func(options *s3.Options) {
					options.Region = string(region)
				},
			)
			return
		}
	} else {
		result = func(filename string, data io.Reader) (err error) {
			var path string
			if strings.HasSuffix(dest, "/") {
				path = dest + filename[strings.LastIndex(filename, "/")+1:]
			} else {
				path = dest
			}
			writeFile, err := os.Create(path)
			if err != nil {
				return
			}
			defer func(open *os.File) {
				_ = open.Close()
			}(writeFile)
			_, err = io.Copy(writeFile, data)
			return
		}
	}
	return
}

func writeConfig(hasAccessKey bool) (err error) {
	if !hasAccessKey {
		return errors.New("writing keys but no keys set")
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dotAwsDir := dir + "/.aws"
	if err = os.MkdirAll(dotAwsDir, 0600); err != nil {
		return
	}
	file, err := os.Create(dotAwsDir + "/credentials")
	if err != nil {
		return
	}
	if err = file.Chmod(0600); err != nil {
		return
	}
	if _, err = file.WriteString(fmt.Sprintf("[default]\naws_access_key_id = %s\naws_secret_access_key = %s\n", Options.AccessKey, Options.SecretKey)); err != nil {
		return
	}
	if err = file.Close(); err != nil {
		return
	}
	file, err = os.Create(dotAwsDir + "/config")
	if err != nil {
		return
	}
	if err = file.Chmod(0600); err != nil {
		return
	}
	if _, err = file.WriteString("[default]\n"); err != nil {
		return
	}
	err = file.Close()
	return
}

func main() {
	err := doEverything()
	if err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
}

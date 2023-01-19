package main

import (
	"context"
	"errors"
	"fmt"
	log "github.com/amoghe/distillog"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jessevdk/go-flags"
	"io"
	"net/url"
	"os"
	"strings"
)

var Commit = "not set"

var Options struct {
	Destination string `short:"d" long:"destination" description:"Setting this makes all the arguments at the end become sources for use with e.g. xargs"`
	Version     bool   `short:"V" long:"version" description:"Print version"`
	Verbose     bool   `short:"v" long:"verbose" description:"Verbose"`
}

var bucketToHost = make(map[string]types.BucketLocationConstraint)

func resolveBucket(ctx context.Context, client *s3.Client, bucket string) (region types.BucketLocationConstraint, err error) {
	region = bucketToHost[bucket]
	if region != "" {
		return
	}
	x, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: aws.String(bucket)})
	if err != nil {
		return
	}

	region = x.LocationConstraint
	bucketToHost[bucket] = region
	return
}

type uploadFunc func(filename string, data io.Reader) (err error)

func doEverything() (err error) {
	args, err := flags.ParseArgs(&Options, os.Args)
	if err != nil {
		return
	}
	if Options.Version {
		log.Errorf("Compiled from %s", Commit)
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

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = "us-west-2"
		//o.UseAccelerate = true
	})

	if len(sources) > 1 && !strings.HasSuffix(dest, "/") {
		return errors.New(fmt.Sprintf(""))
	}

	//s3Client := s3.New(sess)
	//
	//result, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
	//if err != nil {
	//	return
	//}
	//for _, bucket := range result.Buckets {
	//	bucket.
	//
	//}

	var uploadFn uploadFunc
	if strings.HasPrefix(dest, "s3://") {
		var parsed *url.URL
		parsed, err = url.Parse(dest)
		if err != nil {
			return
		}
		if parsed.Path == "" {
			return errors.New(fmt.Sprintf("no path in %s", dest))
		}

		uploadFn = func(filename string, data io.Reader) (err error) {
			var key string
			if strings.HasSuffix(dest, "/") {
				key = parsed.Path + filename[strings.LastIndex(filename, "/")+1:]
			} else {
				key = parsed.Path
			}

			var region types.BucketLocationConstraint
			region, err = resolveBucket(context.Background(), client, parsed.Hostname())
			if err != nil {
				return err
			}

			_, err = client.PutObject(
				context.Background(),
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
		uploadFn = func(filename string, data io.Reader) (err error) {
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

	var successful = true
	for _, source := range sources {
		if strings.HasPrefix(source, "s3://") {
			var parsed *url.URL
			parsed, err = url.Parse(source)
			if err != nil {
				return
			}

			var region types.BucketLocationConstraint
			region, err = resolveBucket(context.Background(), client, parsed.Hostname())
			if err != nil {
				return err
			}

			var output *s3.GetObjectOutput
			getObjectInput := &s3.GetObjectInput{
				Bucket: aws.String(parsed.Hostname()),
				Key:    aws.String(strings.TrimPrefix(parsed.Path, "/")),
			}
			output, err = client.GetObject(
				context.Background(),
				getObjectInput,
				func(options *s3.Options) {
					options.Region = string(region)
				},
			)
			if err != nil {
				log.Errorln(err)
				successful = false
				continue
			}
			err = uploadFn(source, output.Body)
			if err != nil {
				log.Errorln(err)
				successful = false
			}
			err = output.Body.Close()
			if err != nil {
				return
			}
		} else {
			var file *os.File
			file, err = os.Open(source)
			if err != nil {
				return
			}
			err = uploadFn(source, file)
			if err != nil {
				return
			}

			err = file.Close()
			if err != nil {
				return
			}
		}
	}
	if !successful {
		os.Exit(1)
	}
	return
}

func main() {
	err := doEverything()
	if err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
}

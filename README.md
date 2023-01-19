# s3cp [![CircleCI](https://circleci.com/gh/SourceForgery/s3cp.svg?style=svg)](https://circleci.com/gh/SourceForgery/s3cp)

Simple copy util for copying multiple files to/from s3

## Usage

Multiple sources are allowed, but the destination must be a directory (or S3 path prefix) indicated by adding a trailing `/` to the destination path

```
s3cp <SOURCE>... <DEST>
```

| Command line                             | Copy tasks done                                                   |
| ---------------------------------------- | ----------------------------------------------------------------- |
| s3cp s3://foo/bar foobar                 | s3://foo/bar -> ./foobar                                          |
| s3cp s3://foo/bar ./                     | s3://foo/bar -> ./bar                                             |
| s3cp s3://foo/bar .                      | fails because `.` is a directory. Append `/` to get it to work    |
| s3cp s3://foo/bar/baz ./                 | s3://foo/bar/baz -> ./baz                                         |
| s3cp s3://foo/bar s3://foo/baz/baz ./    | s3://foo/bar -> ./bar<br>s3://foo/baz/baz -> ./baz                |
| s3cp bar baz/baz s3://foo/bar/           | ./foo/bar -> s3://foo/bar/bar<br>baz/baz -> s3://foo/bar/baz     |
| s3cp s3://foox/bar baz/baz s3://foo/bar/ | s3://foox/bar -> s3://foo/bar/bar<br>baz/baz -> s3://foo/bar/baz |


## Download the latest

(aka the section that was needed because I was lazy)

1. Click the Circle-CI logo at the top
2. Click the `ci` under workflow of the top build
3. Click `build-and-test-amd64` (only one available right now)
4. Click `ARTIFACTS`
5. Click `s3cp.amd64`

## Build yourself

To get started, download golang https://go.dev/doc/install and run the following commands
```
git clone https://github.com/SourceForgery/s3cp
cd s3cp
go build
./s3cp --help
```

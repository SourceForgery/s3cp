# s3cp [![CircleCI](https://circleci.com/gh/circleci/circleci-docs.svg?style=svg)](https://circleci.com/gh/circleci/circleci-docs)

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

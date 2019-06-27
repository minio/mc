# For maintainers only

## Responsibilities

Please go through this link [Maintainer Responsibility](https://gist.github.com/abperiasamy/f4d9b31d3186bbd26522)

### Setup your mc Github Repository

Fork [mc upstream](https://github.com/minio/mc/fork) source repository to your own personal repository.

```

$ mkdir -p $GOPATH/src/github.com/minio
$ cd $GOPATH/src/github.com/minio
$ git clone https://github.com/$USER_ID/mc
$ 

```

``mc`` uses [govendor](https://github.com/kardianos/govendor) for its dependency management.

### To manage dependencies

#### Add new dependencies

  - Run `go get foo/bar`
  - Edit your code to import foo/bar
  - Run `govendor add foo/bar` from top-level folder

#### Remove dependencies 

  - Run `govendor remove foo/bar`

#### Update dependencies

  - Run `govendor remove +vendor`
  - Run to update the dependent package `go get -u foo/bar`
  - Run `govendor add +external`

### Making new releases 

`mc` doesn't follow semantic versioning style, `mc` instead uses the release date and time as the release versions.

`make release` will generate new binary into `release` directory.

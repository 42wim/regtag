# regtag

Tags docker images directly on a docker registry without pulling/pushing those images.
* Uses your docker login configuration by default.
* Specify -creds username:password for alternative credentials.

Useful for CI/CD for promoting quality to production.

# Binaries

You can find the binaries [here](https://github.com/42wim/regtag/releases/latest)

# Examples
## Using automatic or no credentials
Uses your docker config if exists, otherwise without auth
```
$ regtag registry.mydomain.tld/42wim/matterbridge:1.0 production
production added to https://registry.mydomain.tld/42wim/matterbridge:1.0
```

## Specified credentials
```
$ regtag -creds root:s3cret registry.mydomain.tld/42wim/matterbridge:1.0 production
production added to https://registry.mydomain.tld/42wim/matterbridge:1.0
```

# Building

Go 1.10+ is required 
Make sure you have [Go](https://golang.org/doc/install) properly installed, including setting up your [GOPATH] (https://golang.org/doc/code.html#GOPATH)

```
cd $GOPATH
go get github.com/42wim/regtag
```

You should now have regtag binary in the bin directory:

```
$ ls bin/
regtag
```

# Usage

```
Usage of ./regtag:
./regtag registry/image:tag extratag (uses docker login credentials by default)
  -creds string
        use [username[:password]] for accessing the registry
```

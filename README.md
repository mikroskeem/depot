# Depot

[![Build Status](https://travis-ci.org/mikroskeem/depot.svg?branch=master)](https://travis-ci.org/mikroskeem/depot)

Tired of heavy Maven repository software solutions doing too much or being not straightforward to configure? Me too.

## Building

Grab yourself a recent Go, 1.12+ preferably and run `go build`

## Configuration

See config.sample.toml, comments should guide you enough

## Running

Run built binary directly, make sure that config.toml is in its working directory or supplied via `-config` argument

### Tips

#### Booting with system

If you want to start Depot on system startup, you have two choices:
1) Use systemd/openrc/whatever init system - don't forget to run it as separate user as Depot does not require root privileges!
2) Use Docker - least preferred, but also works

#### Reverse proxy

You should always reverse proxy Depot behind nginx (or other awesome software) and HTTPS, especially
if you have access/deploy credentials defined and host confidential artifacts.

## Credits
- [@kashike](https://github.com/kashike) for generating a name for this project

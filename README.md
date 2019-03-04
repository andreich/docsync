# docsync

`docsync` is a simple utility to sync a local directory to Google Storage.

[![Build Status](https://travis-ci.org/andreich/docsync.svg?branch=master)](https://travis-ci.org/andreich/docsync)
[![Coverage Status](https://coveralls.io/repos/github/andreich/docsync/badge.svg?branch=master)](https://coveralls.io/github/andreich/docsync?branch=master)

## Installation

First set up a configuration file (see sample configuration below). Enable on
Google Cloud Console the [Google Storage JSON API](https://cloud.google.com/storage/docs/json_api/v1/how-tos)
and [Stackdriver Monitoring API](https://cloud.google.com/monitoring/api/enable-api)
and have a service account with the [*Storage Object Creator*](https://cloud.google.com/iam/docs/understanding-roles#storage-roles),
*Storage Object Viewer* and [*Monitoring Metric Writer*](https://cloud.google.com/monitoring/access-control)
 roles.

```sh
$ go install http://github.com/andreich/docsync/cli/docsync
$ curl https://raw.githubusercontent.com/andreich/docsync/master/systemd/install.sh | bash
$ systemctl enable docsync-${USER}
$ systemctl start docsync-${USER}
```

## Sample configuration

```json
{
    "aes_passphrase": "-- password --",
    "bucket_name": "-- bucket --",
    "credentials": {
        "-- copy paste the content from Google credentials JSON file -- "
    },
    "dirs": {
        "-- local directory --": "-- remote directory --"
    },
    "include": [
        "-- file patterns to match --",
        ".*\\.pdf"
    ],
    "interval": "30m",
    "manifest_file": "-- manifest file - not actually used yet, but required --",
    "mover": {
        "from": [
            "-- local directory - I personally use Downloads --"
        ],
        "rules": [
            {
                "patterns": [
                    "-- pattern to match within the content --"
                ],
                "to": "-- local directory to move to - should appear in the top level dirs but it's not required --"
            }
        ]
    },
    "remote_manifest_file": "-- remove manifest file --"
}
```

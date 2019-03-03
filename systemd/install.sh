#!/bin/bash
DOCSYNC=`which docsync`

if [[ -z "$DOCSYNC" ]]; then
    echo "docsync not found in \$PATH"
    echo "Consider installing it with:"
    echo "$ go install github.com/andreich/docsync"
    exit 1
fi

sudo tee /etc/systemd/system/docsync-${USER}.service <<EOF
[Unit]
Description=docsync service - http://github.com/andreich/docsync
After=network-online.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=1
User=${USER}
ExecStart=${DOCSYNC} -config /home/${USER}/.docsync/config.json -dry_run=false

[Install]
WantedBy=multi-user.target
EOF

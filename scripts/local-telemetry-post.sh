#!/bin/bash

# Create a custom telemetry configuration to only log locally
echo "Creating custom telemetry configuration for $1"
mkdir -p $1/etc/telemetrics/

cp $1/usr/share/defaults/telemetrics/telemetrics.conf \
   $1/etc/telemetrics/telemetrics.conf

sed -i -e '/server=/s/clr.telemetry.intel.com/localhost/' \
    -e '/spool_process_time/s/=120/=300/' \
    -e '/record_retention_enabled/s/=false/=true/' \
    $1/etc/telemetrics/telemetrics.conf

exit 0

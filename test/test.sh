#!/bin/sh

HERE=$(dirname $0)

smbd -iFS -s $HERE/smb.conf

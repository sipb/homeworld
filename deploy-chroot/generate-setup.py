#!/usr/bin/env python3

import argparse
import os.path

if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('subnet_byte', help='A unique-per-host byte for which private network to use.')
    args = p.parse_args()

    if not (0 <= int(args.subnet_byte) < 256):
        raise ValueError('subnet_byte must be a byte')

    with open(os.path.expandvars('$HOMEWORLD_DIR/setup.yaml.in')) as fin, open(os.path.expandvars('$HOMEWORLD_DIR/setup.yaml'), 'w') as fout:
        fout.write(fin.read().format(x=args.subnet_byte))

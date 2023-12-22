#!/bin/sh

if [ -z "MASTER_KEY_ADDRESS" ]; then
  echo "MASTER_KEY_ADDRESS env var not set"
  exit 1
fi

echo "Starting node with master key address $MASTER_KEY_ADDRESS"

cp /node/keys/$MASTER_KEY_ADDRESS/master.key /tmp

thor --config-dir=/tmp --network /node/config/genesis.json --bootnode enode://e32e5960781ce0b43d8c2952eeea4b95e286b1bb5f8c1f0c9f09983ba7141d2fdd7dfbec798aefb30dcd8c3b9b7cda8e9a94396a0192bfa54ab285c2cec515ab@10.5.0.2:55555

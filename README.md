# evm-saver-svc

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Decentralized Oracle service to observe Rarimo bridge smart contract deposit events and submit them to Rarimo core.

## Configuration
The following configuration .yaml file should be provided to launch your oracle:

```yaml
log:
  disable_sentry: true
  level: debug

# Port listen requests on
listener:
  addr: :8000

# EVM bridge contract configuration
evm:
  contract_addr: "0xcbc1...df785D12bE"
  rpc: "wss://goerli.infura.io/ws/v3/c29...9"
  start_from_block: # zero if from current
  block_window: # amount of blocks should appear before event becomes fetched
  network_name: Goerli # according to Rarimo chain config 

broadcaster:
  addr: "broadcaster:80"
  sender_account: "rarimo1g...ztx"

core:
  addr: tcp://validator:26657

cosmos:
  addr: validator:9090

subscriber:
  min_retry_period: 10s
  max_retry_period: 10s

profiler:
  enabled: true
  addr: :8080

```

You will also need some environment variables to run:

```yaml
- name: KV_VIPER_FILE
  value: /config/config.yaml # The path to your config file
```

## Run
To start the service (in vote mode) use the following command:
```shell
evm-saver-svc run voter
```

To run in full mode:
```shell
evm-saver-svc run all
```
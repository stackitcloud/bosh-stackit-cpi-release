# STACKIT Cloud Provider Interface (CPI)

This repository contains the BOSH Cloud Provider Interface (CPI) implementation for STACKIT.

## Overview

The STACKIT CPI enables BOSH to provision and manage resources on the STACKIT cloud platform.

## Development

### Prerequisites

- Go 1.24.1 or higher
- Docker (for container builds)
- BOSH CLI

### Building

To build the CPI binary:

```bash
GOFLAGS=-mod=mod go build -o ../../bin/stackit-cpi .
```

This will create a binary in the `bin/` directory.

### Testing

To run the tests:

```bash
GOFLAGS=-mod=mod go test -p 12 -v ./cpi ./cpi/lib
```

To run integration tests:

```bash
cd cpi/integration
GOFLAGS=-mod=mod go run github.com/onsi/ginkgo/v2/ginkgo -p -v --flake-attempts 3 --randomize-all --fail-fast
```

## Configuration

Details about configuring the CPI can be found in the documentation.

## Examples

<!-- Examples will be added here as per project requirements -->

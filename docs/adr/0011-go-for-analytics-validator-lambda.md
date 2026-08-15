# ADR 0011: Use Go for backend Lambdas

Date: 2026-08-15

## Status

Accepted

## Context

ADR 0006 introduced an HTTP API and validator Lambda for analytics ingestion.
The validator is on the request path for every post view event:

```text
CloudFront
  -> API Gateway HTTP API
  -> validator Lambda
  -> SNS
```

The first backend Lambda implementations used Python 3.12. After adding post
view counters, the request-path validator remained intentionally small: parse
the `{slug+}` route parameter, publish `{"post_slug": "..."}` to SNS, and return
`202`.

Measured Lambda report rows from the Python implementation at 1024 MB memory and Go replacement
showed the validator was a good fit for Go:

```text
Python cold-ish: 656.50 ms duration, 741 ms billed, 92 MB max memory
Python warm:      18.04 ms duration,  19 ms billed, 92 MB max memory

Go cold-ish:     133.24 ms duration, 205 ms billed, 37 MB max memory
Go warm:          11.86 ms duration,  12 ms billed, 37 MB max memory
```

The Go validator was also tested at smaller memory sizes:

```text
128 MB cold-ish: 1299.71 ms duration, 1366 ms billed, 34 MB max memory
128 MB warm:       21.26 ms duration,   22 ms billed, 34 MB max memory

256 MB cold-ish:  624.80 ms duration,  705 ms billed, 33 MB max memory
256 MB warm:       16.50 ms duration,   17 ms billed, 33 MB max memory

512 MB cold-ish:  202.95 ms duration,  257 ms billed, 38 MB max memory
512 MB warm:       14.64 ms duration,   15 ms billed, 38 MB max memory
```

## Decision

Use Go for backend Lambdas and deploy them as custom `provided.al2023` runtimes
on `arm64`.

Use 512 MB memory for backend Lambdas unless CloudWatch metrics show a specific
function needs a different setting.

Terraform owns Lambda infrastructure and configuration. Lambda code deployment
is handled by the backend deploy script so code-only changes do not show up as
Terraform infrastructure changes.

The analytics validator was migrated first because it was already measured on
the request path. The aggregate worker and read-side views Lambda are also Go
Lambdas, so the backend Lambda runtime is now consistent across functions.

## Rationale

The backend Lambdas are small, IO-bound functions with narrow AWS SDK usage. Go
reduces cold start duration, billed duration, and memory use without making this
business logic more complex.

Using one Lambda runtime also removes the need to maintain parallel Python and
Go backend test/build paths.

For the validator, 512 MB was the best right-sized baseline from the tested
values. 128 MB and 256 MB used less configured memory but had much slower cold
starts. The function only used about 33-38 MB at runtime, but Lambda CPU scales
with memory, so configured memory still materially affected latency.

## Consequences

The request-path validator has lower cold start latency and lower memory use.
The aggregate worker and read-side views Lambda should be checked with
CloudWatch metrics after deployment to confirm their actual latency and memory
profiles.

512 MB is the default memory target, not a permanent ceiling. Revisit it with
CloudWatch duration, billed duration, and max memory metrics after each Lambda
is migrated.

The repository now needs Go tooling for backend validation and deployment. CI
builds and tests Go Lambdas, packages them, and deploys code with
`aws lambda update-function-code`.

Local backend code deployment requires running:

```sh
deploy/scripts/build-backend-lambdas.sh
deploy/scripts/deploy-backend-lambdas.sh
```

Terraform still needs a built `bootstrap` file when creating a Lambda for the
first time, but after creation it ignores `filename` and `source_code_hash` so
code-only changes do not create Terraform plan noise.

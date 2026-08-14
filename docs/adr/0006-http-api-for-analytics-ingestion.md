# ADR 0006: Use API Gateway HTTP API for blog analytics ingestion

Date: 2026-08-14

## Status

Accepted

## Context

The blog is an Astro static site deployed to private S3 and served publicly
through CloudFront. Earlier decisions keep the public reading path independent
from a running application server, while leaving room for a serverless backend
for metrics.

The first backend capability is blog analytics. It needs to record post view
events, update aggregated counters for the application, and retain detailed
events for later analysis.

The target backend flow is:

```text
CloudFront
  -> API Gateway HTTP API
  -> validator Lambda
  -> SNS
  -> SQS -> worker Lambda -> DynamoDB aggregate post views
  -> Firehose
  -> S3 detailed analytics
  -> Athena queries
```

The analytics backend has two storage needs:

- DynamoDB stores aggregated counters such as post views.
- S3 stores detailed analytics events for later Athena queries.

## Decision

Use API Gateway HTTP API as the analytics ingestion endpoint.

CloudFront remains the public entry point for the site. It will route a
dedicated analytics path to HTTP API, while static assets continue to come from
the private S3 site bucket.

HTTP API will invoke a validator Lambda. The validator Lambda will validate the
incoming analytics event and publish accepted events to SNS. Downstream
consumers will update DynamoDB aggregate post view data and deliver detailed
analytics events to S3 through Firehose.

## Rationale

HTTP API fits this workload because analytics ingestion needs a small HTTPS
endpoint, Lambda integration, and low operational overhead. REST API would add
features this path does not currently need. An always-on backend service would
add runtime maintenance and cost without improving the static reading path.

Putting HTTP API behind CloudFront keeps the browser-facing boundary on the
same hostname and distribution pattern already used by the site. The static
frontend can emit analytics events without depending on a separate public
application server.

The validator Lambda keeps trust-boundary checks close to ingress. SNS decouples
request acceptance from analytics processing so page views do not wait for
DynamoDB aggregation or S3 delivery.

DynamoDB is a better fit than Athena for fast aggregate counters. S3 and Athena
are a better fit than DynamoDB for detailed event history because raw analytics
events can be stored cheaply and queried when needed.

## Consequences

The deployed website still has no runtime dependency on the homelab or a Node
server. If the analytics backend is unavailable, the static site can continue
serving, but analytics events may be rejected or lost depending on the failure.

HTTP API should stay narrow. New analytics events should be added by extending
the validated event schema, not by adding unrelated application endpoints to
this API.

Detailed event data can be queried later with Athena from S3, while DynamoDB
serves fast aggregate counters for the application.

Terraform will manage the app-specific AWS resources for this backend in this
repository, consistent with the existing infrastructure-as-code decision.

# ADR 0001: Use Astro for the static frontend on AWS

Date: 2026-08-11

## Status

Accepted

## Context

This project is a content-focused blog. The frontend should be fast,
SEO-friendly, cheap to host, and simple to operate.

The blog may also have a serverless backend for metrics, but the public reading
experience should not depend on a running application server.

## Decision

Use Astro to build the frontend as a static site. Astro is a good fit because
it generates SEO-friendly HTML pages and supports Markdown content, which keeps
blog posts easy to write and review in Git.

Astro will generate the website into `dist/`. The generated files will be
uploaded to a private S3 bucket and served publicly through CloudFront:

```text
Astro source
  -> astro build
  -> dist/
  -> private S3 bucket
  -> CloudFront
  -> Website
```

S3 stores the built assets. CloudFront is the public entry point, handles HTTPS,
and caches the site close to readers. The S3 bucket stays private instead of
being exposed as a public website bucket.

DNS for `blog.mrembiasz.pl` is managed outside AWS. Terraform manages the ACM
certificate request in `us-east-1` for CloudFront, but the DNS validation CNAME
and the final hostname record must exist in the external DNS provider.

## Rationale

Astro fits the main workload: mostly static content, Markdown posts, and pages
that should be easy for crawlers to read.

A server-rendered frontend would add a runtime service to operate without
improving the normal reading path. A client-heavy single page app would make
SEO and content delivery more complicated than needed for a blog.

S3 and CloudFront keep hosting simple. S3 is durable object storage for the
built files, while CloudFront provides the public HTTPS endpoint, caching, and a
clean boundary around the private bucket.

Keeping DNS outside AWS avoids moving the existing domain authority into this
application repository. The tradeoff is that ACM validation and hostname
cutover require an explicit DNS change outside Terraform.

## Consequences

The deployed website has no runtime dependency on the homelab or a Node server.
If CI/CD is unavailable, publishing new content is delayed, but the already
deployed site keeps serving from AWS.

Astro keeps the frontend small while still supporting blog features such as
Markdown content pages, metadata, RSS, and SEO-friendly HTML.

CloudFront certificate creation depends on external DNS validation. If the ACM
validation CNAME is missing or changed, Terraform apply will wait or fail until
DNS is corrected.

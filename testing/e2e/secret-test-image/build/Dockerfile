FROM alpine:3.18.6

# `test -s` fails when the secret file is absent or empty, so a build that
# succeeds proves the --secret reached BuildKit with content in it. This
# deliberately does NOT cat or copy the secret: buildx output is streamed to
# stderr, so printing it would put it in the CI log.
#
# There is deliberately no `# syntax=docker/dockerfile:1` directive here.
# RUN --mount=type=secret is supported by BuildKit's built-in frontend, and
# skipping the directive avoids pulling the frontend image from Docker Hub on
# every CI run - registry rate limits are a recurring pain in this repo.
RUN --mount=type=secret,id=dockem_e2e_secret test -s /run/secrets/dockem_e2e_secret

COPY message /tmp/message
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD ["/bin/cat /tmp/message"]

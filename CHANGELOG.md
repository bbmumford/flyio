# Changelog

## 0.1.0 (2026-11-26)


### Features

* a Fly.io deploy adapter: configuration, SSL certificate renewal through the Fly API, and a health endpoint ([e5d2474](https://github.com/bbmumford/flyio/commit/e5d24743c713cef5991718034938791a035652f0))


### Bug Fixes

* a certificate renewal counts only when the API reports success ([0f7c9c6](https://github.com/bbmumford/flyio/commit/0f7c9c6507ec9459c1a1a71f700897c7605ed674))
* go.mod requires argus v0.1.0, buildinfo v0.1.0, cfgload v0.2.0 and envreq v0.0.2 with no local replaces, and Go 1.27.4 ([e089463](https://github.com/bbmumford/flyio/commit/e08946342951506d67fe952a2d1db60c08b6883f))
* StartAutoRenew logs failures when no error callback is given ([b189980](https://github.com/bbmumford/flyio/commit/b189980c1c11a478a0230e02a760e52620089e57))

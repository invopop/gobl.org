# gobl.org

Public facing GOBL website, built with [Hugo](https://gohugo.io) (extended, v0.169.0).

## Coverage data

The homepage lists the tax regimes, addons, and conversion formats from the
`data/` directory. Refresh them from the [GOBL.dev](https://gobl.dev) API with:

```
mage -v data
```

Run this when upgrading to a new GOBL version, alongside the schema below. Set
`GOBL_DEV_URL` to read from another instance, such as a local `gobl serve`:

```
GOBL_DEV_URL=http://localhost:8080 mage -v data
```

## Schema

Run:

```
mage -v schema
```

To copy the schema from the GOBL project in the path `../gobl`.

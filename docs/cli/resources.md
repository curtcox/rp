# rp resources / rp resource

Inspect the resources declared in a project's config.

```
rp resources
rp resource NAME
```

## rp resources — list

Prints one line per resource: `name`, `type`, and the URI of its first
realization, tab-separated and sorted by name.

<!-- rp-example: id=resources-list cwd=fixture status=ready -->
```console
$ rp resources
bug_report	BugReport	file://bug.md
repo	GitRepo	file://.
```

## rp resource NAME — inspect one

Prints the full resource (type plus every realization) as JSON.

<!-- rp-example: id=resource-show cwd=fixture status=ready -->
```console
$ rp resource repo
{
  "type": "GitRepo",
  "realizations": [
    {
      "id": "repo.local",
      "kind": "local_path",
      "uri": "file://.",
      "media_type": "inode/directory"
    }
  ]
}
```

## Adding resources

Resources are usually declared in `.rp/` YAML (see the
[config reference](../config/reference.md)), but you can also add one from the
command line:

<!-- rp-example: id=resource-add cwd=fixture status=todo -->
```console
$ rp add resource notes --type Document --file notes.md --media-type text/markdown
# (output to be captured)
```

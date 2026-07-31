<!--
Thanks for contributing to GoScouter!

Please read CONTRIBUTING.md if you haven't yet:
https://github.com/GoScouter/goscouter/blob/main/CONTRIBUTING.md
-->

## Summary

<!-- What does this change do, and why? One or two paragraphs. -->

## Related issues

<!-- e.g. "Closes #123", "Part of #45". Write "None" if this isn't tracked. -->

## Type of change

<!-- Check every box that applies, and add the matching type/* label. -->

- [ ] Bug fix (`type/bug`)
- [ ] New feature (`type/feature`)
- [ ] Enhancement to existing behavior (`type/enhancement`)
- [ ] Refactor, no behavior change (`type/refactor`)
- [ ] Performance (`type/performance`)
- [ ] Documentation (`type/documentation`)
- [ ] Chore / maintenance (`type/chore`)

## Breaking changes

<!--
Does this change CLI flags, output formats, or the module protocol in a way
that breaks existing users or modules? If so, describe the break and the
migration path. Otherwise write "None".
-->

None

## How this was tested

<!--
Which tests did you add, and how did you verify the change by hand?
Include the command you ran and the platform you ran it on.
-->

```
$ ./bin/gs --target https://example.com
```

- Platform tested:  <!-- e.g. linux/amd64, darwin/arm64 -->

## Checklist

- [ ] I read [CONTRIBUTING.md](../CONTRIBUTING.md)
- [ ] The branch is named `<type>/<short-description>` and commits follow
      [Conventional Commits](https://www.conventionalcommits.org/)
- [ ] Documentation (README, doc comments) is updated where it applies

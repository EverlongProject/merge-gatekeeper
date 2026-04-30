# Action Details

## Action Inputs

<!-- == export: inputs / begin == -->

| Name       | Description                                                                                                                                                                                                                                                                                          | Required |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------: |
| `token`    | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                                                                                                                                                                                                                            |   Yes    |
| `self`     | The name of Merge Gatekeeper job, and defaults to `merge-gatekeeper`. This is used to check other job status, and do not check Merge Gatekeeper itself. If you updated the GitHub Action job name from `merge-gatekeeper` to something else, you would need to specify the new name with this value. |          |
| `interval` | Check interval to recheck the job status. Default is set to 5 (sec).                                                                                                                                                                                                                                 |          |
| `timeout`  | Timeout setup to give up further check. Default is set to 600 (sec).                                                                                                                                                                                                                                 |          |
| `success-confirmation-polls` | Number of additional consecutive successful polls required after Merge Gatekeeper first sees all validations as green. `0` keeps the current behavior.                                                                                                                                      |          |
| `ignored`  | Jobs to ignore regardless of their statuses. Defined as a comma-separated list. Use this for checks you intentionally do not want to gate merges on, for example `CodeQL`.                                                                                                                            |          |
| `ignore-dynamic-github-workflows` | Ignore check runs whose backing workflow run resolves to a GitHub-managed workflow path under `dynamic/`. This is useful for GitHub-generated workflows such as Copilot review jobs that are not declared in the repository itself. Defaults to `true`; set it to `false` to keep those checks in scope. |          |
| `ref`      | Git ref to check out. This falls back to the HEAD for given PR, but can be set to any ref.                                                                                                                                                                                                           |          |

<!-- == export: inputs / end == -->

## Usage

### Copy Standard YAML

<!-- == export: simple-usage / begin == -->

The easiest approach is to copy the standard definition, and save it under `.github/workflows` directory. There is no further modification required unless you have some specific requirements.

#### With `curl`

```bash
curl -sSL https://raw.githubusercontent.com/EverlongProject/merge-gatekeeper/main/example/merge-gatekeeper.yml \
  > .github/workflows/merge-gatekeeper.yml
```

#### Directly copy YAML

The below is the copy of [`/example/merge-gatekeeper.yml`](/example/merge-gatekeeper.yml), with extra comments.

<!-- == imptr: basic-yaml / begin from: ../example/definitions.yaml#[standard-setup] wrap: yaml == -->
```yaml
---
name: Merge Gatekeeper

on:
  pull_request:
    branches:
      - main
      - master

jobs:
  merge-gatekeeper:
    runs-on: ubuntu-latest
    # Restrict permissions of the GITHUB_TOKEN.
    # Docs: https://docs.github.com/en/actions/using-jobs/assigning-permissions-to-jobs
    permissions:
      actions: read
      checks: read
      statuses: read
    steps:
      - name: Run Merge Gatekeeper
        # NOTE: v1 is updated to reflect the latest v1.x.y. Please use any tag/branch that suits your needs:
        #       https://github.com/EverlongProject/merge-gatekeeper/tags
        #       https://github.com/EverlongProject/merge-gatekeeper/branches
        uses: EverlongProject/merge-gatekeeper@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
```
<!-- == imptr: basic-yaml / end == -->

<!-- == export: simple-usage / end == -->

### Delayed Job Registration

Some CI systems generate child jobs dynamically. In those setups, the parent job can sometimes finish successfully before GitHub has fully registered the child jobs as checks. If that happens, Merge Gatekeeper may briefly observe an all-green state too early.

Use `success-confirmation-polls` to require additional consecutive successful polls after the first all-green result. For example, setting it to `1` means Merge Gatekeeper must see two consecutive successful polls before it returns success. The default `0` keeps the original behavior and should remain the normal setting unless your workflow has this race.

### Ignoring GitHub-Managed Dynamic Workflows

Some checks are created by GitHub itself rather than by workflows committed in the repository. Merge Gatekeeper ignores those checks by default when it resolves a check run back to a workflow run whose path starts with `dynamic/`. Set `ignore-dynamic-github-workflows: false` if you need to keep them in merge evaluation.

CodeQL and other GitHub Advanced Security checks are different. They can appear as GitHub-owned checks without a backing workflow run in the Actions workflow-runs API, so `ignore-dynamic-github-workflows` does not exclude them. If you do not want CodeQL to gate merges, add it to `ignored` explicitly.

Example:

```yaml
jobs:
  merge-gatekeeper:
    steps:
      - name: Run Merge Gatekeeper
        uses: EverlongProject/merge-gatekeeper@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          ignored: CodeQL
```

Lookup failures stay fail-open. Merge Gatekeeper will continue counting the check normally, and the action output will include a `gh api` command you can run to inspect the workflow lookup path directly.

### Using Importer

You can also use the latest spec by using Importer to import directly from the sample setup in this repository.

Create a YAML file with just a single Importer Marker:

```yaml
# == imptr: merge-gatekeeper / begin from: https://github.com/EverlongProject/merge-gatekeeper/blob/main/example/definitions.yaml#[standard-setup] ==
# == imptr: merge-gatekeeper / end ==
```

With that, you can simply run `importer update FILENAME` to get the latest spec. You can also update the file used to specific branch or version.

###

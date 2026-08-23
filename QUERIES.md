# API for GitHub Star List (2026-08-23)

Discussion: <https://github.com/orgs/community/discussions/8293>

## Setup

```bash
gh auth status | grep -qF "'user'" || gh auth refresh -h github.com -s user
```

## `gh starlist list`

```bash
gh api graphql -F first=100 -F after=null -f query='
query ViewerLists($first: Int!, $after: String) {
  viewer {
    lists(first: $first, after: $after) {
      pageInfo { hasNextPage endCursor }
      nodes { id name slug description isPrivate updatedAt items { totalCount } }
    }
  }
}'
```

### `--user USER`

```bash
gh api graphql -f login=USER -F first=100 -F after=null -f query='
query UserLists($login: String!, $first: Int!, $after: String) {
  user(login: $login) {
    lists(first: $first, after: $after) {
      pageInfo { hasNextPage endCursor }
      nodes { id name slug description isPrivate updatedAt items { totalCount } }
    }
  }
}'
```

## `gh starlist view <list>`

```bash
gh api graphql -f listId=$LIST_ID -F first=100 -F after=null -f query='
query ListItems($listId: ID!, $first: Int!, $after: String) {
  node(id: $listId) {
    ... on UserList {
      items(first: $first, after: $after) {
        pageInfo { hasNextPage endCursor }
        nodes {
          ... on Repository {
            nameWithOwner description url stargazerCount isArchived
            primaryLanguage { name }
          }
        }
      }
    }
  }
}'
```

## `gh starlist create <name>`

```bash
gh api graphql -f name='Rust Tools' -f description='Things written in Rust' -F isPrivate=true -f query='
mutation CreateList($name: String!, $description: String, $isPrivate: Boolean) {
  createUserList(input: {name: $name, description: $description, isPrivate: $isPrivate}) {
    list { id name slug description isPrivate }
  }
}'
```

## `gh starlist edit <list>`

```bash
gh api graphql -f listId=$LIST_ID -f name='Rust CLI' -F description=null -F isPrivate=false -f query='
mutation UpdateList($listId: ID!, $name: String, $description: String, $isPrivate: Boolean) {
  updateUserList(input: {listId: $listId, name: $name, description: $description, isPrivate: $isPrivate}) {
    list { id name slug description isPrivate }
  }
}'
```

## `gh starlist delete <list>`

```bash
gh api graphql -f listId=$LIST_ID -f query='
mutation DeleteList($listId: ID!) {
  deleteUserList(input: {listId: $listId}) {
    user { login }
  }
}'
```

## `gh starlist add <list> <repository>`

### 1. Look up the repository's node ID and star state

```bash
gh api graphql -f owner=OWNER -f name=REPO -f query='
query RepoRef($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) { id nameWithOwner viewerHasStarred }
}'
```

### 2. Walk every list to find the current membership

Loop over `ViewerLists` plus `ListItems` for each list. There is no API for the reverse lookup from the repository side.

### 3. Replace the membership set with "current set + target list"

```bash
gh api graphql -f itemId=$REPO_ID -f listIds[]=$LIST_ID_A -f listIds[]=$LIST_ID_B -f query='
mutation SetLists($itemId: ID!, $listIds: [ID!]!) {
  updateUserListsForItem(input: {itemId: $itemId, listIds: $listIds}) {
    lists { id }
  }
}'
```

### 4. Star the repository if it is not starred yet (skipped with `--no-star`)

```bash
# Option 1: REST
gh api --method PUT --silent user/starred/OWNER/REPO

# Option 2: GraphQL
gh api graphql -f starrableId=$REPO_ID -f query='
mutation Star($starrableId: ID!) {
  addStar(input: {starrableId: $starrableId}) { starrable { id } }
}'
```

## `gh starlist remove <list> <repository>`

Same steps 1-3 as `add`, but pass the set with the target list's ID removed from `listIds` in step 3.
Step 4 is not performed, since the star is left in place.

To empty a list, pass `[]` as `listIds`.

```bash
gh api graphql -f itemId=$REPO_ID -f listIds[]=$LIST_ID_A -f query='
mutation SetLists($itemId: ID!, $listIds: [ID!]!) {
  updateUserListsForItem(input: {itemId: $itemId, listIds: $listIds}) {
    lists { id }
  }
}'
```

## `gh starlist export`

At first `ViewerLists` / `UserLists`, then `ListItems` for each list, plus the starred list, to also emit starred repositories that belong to no list.

```bash
# Option 1: REST
gh api --paginate -H 'Accept: application/vnd.github.star+json' \
  'user/starred?per_page=100&sort=created&direction=desc'

# Option 2: GraphQL
gh api graphql -F first=100 -F after=null -f query='
query ViewerStarred($first: Int!, $after: String) {
  viewer {
    starredRepositories(first: $first, after: $after, orderBy: {field: STARRED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      edges {
        starredAt
        node {
          nameWithOwner description url stargazerCount isArchived
          primaryLanguage { name }
        }
      }
    }
  }
}'
```

### `--user USER`

```bash
# Option 1: REST
gh api --paginate -H 'Accept: application/vnd.github.star+json' \
  'users/USER/starred?per_page=100&sort=created&direction=desc'

# Option 2: GraphQL
gh api graphql -f login=USER -F first=100 -F after=null -f query='
query UserStarred($login: String!, $first: Int!, $after: String) {
  user(login: $login) {
    starredRepositories(first: $first, after: $after, orderBy: {field: STARRED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      edges {
        starredAt
        node {
          nameWithOwner description url stargazerCount isArchived
          primaryLanguage { name }
        }
      }
    }
  }
}'
```

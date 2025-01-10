#!/bin/sh
set -e

# Determine's the source level of a GitHub repository/commit/branch.
# Usage:
# ./determine_source_level_gh.sh <COMMIT> <REPO> <BRANCH>

# TODO: Commit is currently ignored, do we really need it
COMMIT=$1
REPO=$2
BRANCH=$3

# TODO: Add GitHub token for non-public repos?
GITHUB_RULESET=$(curl -s -L \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/repos/${REPO}/rules/branches/${BRANCH})

# Check if response is valid JSON
if ! echo $GITHUB_RULESET | jq '.' >/dev/null 2>&1; then
    echo "Error: Invalid response from GitHub API"
    echo "$GITHUB_RULESET"
    exit 1
fi

# Check if response is empty array
if [ "$GITHUB_RULESET" = "[]" ]; then
    echo "SLSA_SOURCE_LEVEL_1"
    exit 0
fi

# Check continuity requirement
# We'll assume it meets this requirement if the branch prevents deletions
# and force pushes.
# TODO: Should other things be checked too?
DELETION_RULESET_ID=$(echo $GITHUB_RULESET | jq '.[] | select(.type=="deletion").ruleset_id')
NO_FORCE_PUSH_RULESET_ID=$(echo $GITHUB_RULESET | jq '.[] | select(.type=="non_fast_forward").ruleset_id')

# Do they both exist and have rulesets?
if [ -z "${DELETION_RULESET_ID}" ] || [ -z "${NO_FORCE_PUSH_RULESET_ID}" ] ; then
    echo "SLSA_SOURCE_LEVEL_1"
    exit 0
fi

# TODO:
# * Check if the needed rules are active.
# * Check that they've been active for the last X days
# * Use "updated_at" for the time rather than created
#   * Folks can disable enforcement and change other important properties
#     without changing
# * Only run this script for 'push' events to 

# Is the deletion rule active?
DELETION_RULESET=$(curl -L \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/repos/${REPO}/rulesets/${DELETION_RULESET_ID})

ENFORCEMENT_STATUS=$(echo $DELETION_RULESET | jq '.enforcement')
if [ "${ENFORCEMENT_STATUS}" = 'active' ]


SOURCE_LEVEL="SLSA_SOURCE_LEVEL_1"

if [ "$NO_DELETION" = "true" ] && [ "$NO_FORCE_PUSH" = "true" ]; then
    SOURCE_LEVEL="SLSA_SOURCE_LEVEL_2"
fi
echo $SOURCE_LEVEL
# ext-build-info

## About this plugin
The ext-build-info plugin collects issues from git comment messages and branch names and validates that they are known by the tracker. 

## Installation with JFrog CLI
Installing the latest version:

`$ jf plugin install ext-build-info`

Installing a specific version:

`$ jf plugin install ext-build-info@version`

Uninstalling a plugin

`$ jf plugin uninstall ext-build-info`

## Usage
### Commands
* collect-issues
  - Arguments:
    - build name - The name of the build.
    - build number - The number of the build.
    - path to .git - Path to a directory containing the .git directory. If not specified, the .git directory is assumed to be in the 
      current directory or in one of the parent directories.
  - Flags
    - --server-id - [Optional] Server ID configured using the config command.
    - --project - [Optional] Server ID configured using the config command.
    - --tracker - [Optional] Tracker to use to collect related issue from.
    - --tracker-url - [Optional] Tracker base url to use to collect related issue from.
    - --tracker-username - [Optional] Tracker username to use to collect related issue from.
    - --tracker-token - [Optional] Tracker token to use to collect related issue from.
    - --regexp - [Optional] A regular expression used for matching the git commit messages.
    - --key-group-index - [Default: 1] The capturing group index in the regular expression used for retrieving the issue key.
    - --aggregate - [Default: false] Set to true, if you wish all builds to include issues from previous builds.
    - --aggregation-status - [Optional] If aggregate is set to true, this property indicates how far in time should the issues be 
      aggregated. In the above example, issues will be aggregated from previous builds, until a build with a RELEASE status is found. 
      Build statuses are set when a build is promoted using the jf rt build-promote command. 

  - Example:
    ```
    $ jf ext-build-info collect-issues --tracker=Jira MyBuild 1

    [Info] Reading the git branch, revision and remote URL and adding them to the build-info.
    [Info] Collecting build issues from VCS...
    [Info] Collected 1 issue details for MyBuild/1.
    ```
* clean-slate
  - Arguments:
    - build name - The name of the build.
    - build number - The number of the build.
  - Flags
    - --project - [Optional] Server ID configured using the config command.

  - Example:
    ```
    $ jf ext-build-info clean-slate MyBuild 1

    [Info] Clearing all existing build-info to start from a clean slate.
    [Info] Removing build-info directory: /path/to/build/info
    ```

### Environment variables
The plugin can lookup the tracker url, username and token using the JFrog Pipelines integration environment variables.  

## Additional info
None.

## Release Notes
The release notes are available [here](RELEASE.md).

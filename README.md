# AWS List

A simple utility to export an overview of running AWS EC2 instances.

The following information will be displayed for each running instance:

* Account ID
* Instance ID
* Instance Name
* AMI ID
* Age of AMI in days

This can be helpful for making sure that instances are running up to date AMIs, especially when you have a lot of accounts to deal with.


## Usage

Once installed, from the download page, or after building from source, simply run the tool:

```bash
aws-list
```

By default it will use the AWS environmental variables, or the contents of `~/.aws/credentials` as you would expect.

If you have access to multiple roles you can pass a file containing them to the tool:

```bash
aws-list -roles=roles.list
```

The roles.list file would contain something like this:

```
arn:aws:iam::123456789091:role/account-name-ABCDFFDFDKLJ
arn:aws:iam::123456789092:role/dns-account-name-XYZAXYDSFLKJ
```

## Issues?

Please report.

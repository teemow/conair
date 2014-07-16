# Conair - an opinionated Docker clone

Without these tools you are probably screwed. PRs are welcome!

 * archlinux
 * systemd-nspawn
 * systemd-networkd (systemd 215+)
 * btrfs

## Build

Install go and run make. This will install conair to `/usr/local/bin`:

```
make && make install
```

## Usage

Initialize your environment with:
```
conair init
```

Most of the times `conair` requires root privileges. So make sure to prepend `sudo`.

Create a base image:
```
btrfs subvolume create /var/lib/conair/images/base
pacstrap -i -c -d /var/lib/conair/images/base base
```

## Commands

conair init    # Setup a bridge for the containers and add some iptables forwarding
conair destroy # Remove bridge, iptables and unit file
conair images  # List all available conair images
conair run     # Run a container
conair ps      # List all conair containers
conair start   # Start a container
conair stop    # Stop a container
conair status  # Status of container
conair attach  # Attach to container
conair commit  # Commit a container
conair rm      # Remove a container
conair rmi     # Remove an image
conair help    # Show a list of commands or help for one command
conair version # Print the version and exit

## Why?

* For the fun of it
* Proper systemd integration
* No need to use supervisord _if_ you need to run multiple services within the same container
* All CoreOS components in a single container
 * Systemd
 * Fleet
 * Etcd
 * Docker

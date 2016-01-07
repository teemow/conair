# Conair - systemd OS containers

 * Build Archlinux nspawn containers easily.
 * Run other container orchestration tools inside of containers within a second. Much faster than using vagrant.
 * Manage multiple services inside of a container without the hassle of supervisord. 

## Dependencies

 * archlinux/CoreOS
 * systemd-nspawn
 * systemd-networkd (systemd 215+)
 * systemd-machined (systemd 219+)
 * (btrfs)

## Build

Install go and run make. This will install conair to `/usr/local/bin`:

```
make && make install
```

If you don't have a btrfs partition/root then you can create a loopback device with btrfs and mount it to `/var/lib/machines`. I wrote a little tool called [loopback](https://github.com/teemow/loopback) for it.

```
sudo loopback create --name=conair --size=10 /var/lib/machines
```

## Usage

Initialize your environment with:
```
conair init
```

Most of the times `conair` requires root privileges. So make sure to prepend `sudo`.

Create a base image:

```
conair bootstrap base # if you are on archlinux (pacstrap required)
conair pull base     # download an image
```

Or DIY:
```
btrfs subvolume create /var/lib/machines/base
pacstrap -i -c -d /var/lib/machines/base base
```

## Build an image

Dockerfiles and Conairfiles are supported. FROM, RUN and ADD are implemented. Conairfiles support PKG and ENABLE to install pacman packages and enable systemd units.

```
conair build my-new-image
```

## Commands

```
conair init      # Setup a bridge for the containers and add some iptables forwarding
conair destroy   # Remove bridge, iptables and unit file
conair images    # List all available conair images
conair run       # Run a container
conair ps        # List all conair containers
conair start     # Start a container
conair stop      # Stop a container
conair status    # Status of container
conair attach    # Attach to container
conair commit    # Commit a container
conair rm        # Remove a container
conair rmi       # Remove an image
conair pull      # Pull an image
conair bootstrap # Creates an arch rootfs with pacstrap.
conair help      # Show a list of commands or help for one command
conair version   # Print the version and exit
```

## Why?

* Testing different docker versions inside of containers
* Having a nice workflow to create new nspawn containers
* A proper systemd integration between host and containers
* No need to use supervisord _if_ you need to run multiple services within the same container
* All CoreOS components in a single container but change them independently and test quickly
 * Systemd
 * Fleet
 * Etcd
 * Docker

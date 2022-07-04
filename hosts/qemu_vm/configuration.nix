

# Edit this configuration file to define what should be installed on
# your system.  Help is available in the configuration.nix(5) man page
# and in the NixOS manual (accessible by running ‘nixos-help’).

{ config, lib, pkgs, ... }:

{
  fileSystems."/homeworld" = {
    device = "host0";
    fsType = "9p";
    options = [
      "trans=virtio" "version=9p2000.L" "ro" "_netdev"
    ];
  };

  time.timeZone = "America/New_York";

  networking.hostName = "hyades-virt-1";
  services.sshd.enable = true;

  networking.firewall.allowedTCPPorts = [ 22 ];

  users.users.root.password = "root";
  services.openssh.permitRootLogin = lib.mkDefault "yes";
  services.getty.autologinUser = lib.mkDefault "root";
  
  # This value determines the NixOS release from which the default
  # settings for stateful data, like file locations and database versions
  # on your system were taken. It‘s perfectly fine and recommended to leave
  # this value at the release version of the first install of this system.
  # Before changing this value read the documentation for this option
  # (e.g. man configuration.nix or on https://nixos.org/nixos/options.html).
  system.stateVersion = "22.05"; # Did you read the comment?
}

# vim:set ts=2 sw=2 et:

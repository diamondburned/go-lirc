{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:

    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = pkgs.lib;

        module = pkgs.buildGoModule {
          pname = "lirc";
          version = self.rev or "unknown";
          src = ./.;
          vendorHash = "sha256-4WWEfQEI8P9swKrgxFv1QThiV6eqFB7sKZmyQ+C9wwM=";
        };

        vendoredSrc = pkgs.symlinkJoin {
          name = "lirc-vendored";
          paths =
            [ module.src ]
            ++ (
              if module.vendorHash == null then
                [ ]
              else
                [
                  (pkgs.linkFarm "lirc-vendor" [
                    {
                      name = "vendor";
                      path = module.goModules;
                    }
                  ])
                ]
            );
        };

        runTests = pkgs.writeShellApplication {
          name = "lirc-tests";
          text = "cd ${vendoredSrc} && go test -v ./...";
          runtimeInputs = [ pkgs.go ];
        };
      in
      {
        formatter = pkgs.nixfmt-rfc-style;

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            self.formatter.${system}
          ];
        };

        checks.vmTest = pkgs.testers.runNixOSTest {
          name = "lirc-vm-test";
          nodes.machine =
            {
              config,
              pkgs,
              lib,
              ...
            }:
            {
              services.lirc = {
                enable = true;
                configs = [ (builtins.readFile ./testdata/remotes/samsung/BN59-00516A.lircd.conf) ];
                options = ''
                  [lircd]
                  driver = file
                  device = auto
                '';
              };
              environment.variables.LIRC_TEST_UNIX_ADDRESS = config.passthru.lirc.socket;
            };
          testScript = ''
            machine.start()
            machine.wait_for_unit("lircd.socket")
            machine.succeed("${lib.getExe runTests}")
          '';
        };
      }
    );
}

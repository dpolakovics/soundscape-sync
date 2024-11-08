{ pkgs ? import <nixpkgs> {} }:
# with import <nixpkgs> {
#   crossSystem = {
#     config = "x86_64-w64-mingw32";
#   };
# };
# let
#   # pkgs = (import <nixpkgs>{ crossSystem = {config = "x86_64-w64-mingw32";}; });
#   pkgs = import <nixpkgs> {
#     localSystem = "x86_64-linux"; # buildPlatform
#     crossSystem = "x86_64-w64-mingw32"; # Note the `config` part!
#   };
# in
pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    gotools
    gopls
    go-outline
    gopls
    gopkgs
    godef
    golint
    delve
    pkg-config
    clang
    pkg-config
    glib
    cairo
    pango
    gdk-pixbuf
    atk
    ffmpeg
    # linux
    wayland
    wayland-protocols
    xorg.libX11
    xorg.libXcursor
    xorg.libXi
    xorg.libXrandr
    xorg.libXinerama
    libxkbcommon
    libGL
    mesa
    # windows
  ];

  shellHook = ''
    export LD_LIBRARY_PATH=${pkgs.lib.makeLibraryPath [
      pkgs.wayland
      pkgs.libGL
      pkgs.xorg.libX11
      pkgs.xorg.libXcursor
      pkgs.xorg.libXi
      pkgs.xorg.libXrandr
      pkgs.libxkbcommon
    ]}:$LD_LIBRARY_PATH
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin
    export CGO_ENABLED=1
  '';
}

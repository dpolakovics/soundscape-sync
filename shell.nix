{ pkgs ? import <nixpkgs> {} }:
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
    gcc
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
    xorg.libXxf86vm
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
      pkgs.xorg.libXinerama
      pkgs.xorg.libXxf86vm
      pkgs.libxkbcommon
    ]}:$LD_LIBRARY_PATH
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin
    export CGO_ENABLED=1
  '';
}

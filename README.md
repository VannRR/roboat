# Roboat

**Roboat** is a rofi script written in Go for viewing feeds from newsboat.

## Requirements

- A newsboat SQLite cache file (`cache.db`). You can set a custom path with the environment variable `$ROBOAT_CACHE_PATH`.
- Optionally, `xdg-utils` or you can set a browser with the environment variable `$ROBOAT_BROWSER`.

## Installation

### Build from source
1. Clone the repository:
    ```sh
    git clone --depth 1 https://github.com/vannrr/roboat.git
    cd roboat
    ```

2. Build Roboat:
    - a.
    ```sh
    make install
    ```
    - b.
    or if you don't have `make`:
    ```sh
    go build -ldflags="-w -s" -o roboat main.go
    mkdir -p ~/.config/rofi/scripts/
    cp roboat ~/.config/rofi/scripts/
    ```

### Install from binary
1. Download from release page https://github.com/vannrr/roboat/releases/latest
2. place binary in `~/.config/rofi/scripts/`

## Usage

Run the script with rofi:
```sh
rofi -show roboat
```

### Notes

#### Hotkeys (Alt+1, etc.) Not Working
If hotkeys are not working in rofi, check the following properties in the rofi config:
`kb-custom-1`, `kb-custom-2`, and `kb-custom-3`. If they are not set to their default values,
the hotkeys listed in roboat will be incorrect.

## Links

- https://github.com/newsboat/newsboat
- https://github.com/davatorium/rofi
- https://github.com/lbonn/rofi (wayland fork)
- https://github.com/davatorium/rofi/blob/next/doc/rofi-script.5.markdown

## License

This project is licensed under the MIT License. See the LICENSE file for details.

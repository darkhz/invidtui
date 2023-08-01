# Requirements
Before installation, ensure that the required dependencies are installed:
* **MPV**
* **ffmpeg**
* **youtube-dl** or **yt-dlp**

Optionally, **mpv-mpris** can be installed for MPRIS playback control.

# Installation
After installing the dependencies, you can install invidtui using any one method listed below.

## Package manager
If your distribution's repositories have invidtui, you can install it directly with your package manager.
For **Arch Linux**, The package invidtui-bin is in the AUR. Install it using an AUR helper of your choice like so:<br/>
```
<aur-helper> -S invidtui-bin.
```

## Releases
You can retrieve the package's tagged release from the project's [Releases](https://github.com/darkhz/invidtui/releases/) page.

Before downloading, note:
- The latest tag of the release
- Your operating system (for example, Linux)
- Your architecture (for example, x86_64)

The binary is packaged in a gzipped tar file (with the extension `.tar.gz`) in the format:
`invidtui_<tag>_<Operating System>_<Architecture>.tar.gz`

To download a package for:
- with the release tag 'v0.3.2',
- a 'Linux' distribution, 
- on the 'x86_64' architecture, 

You would select:
`invidtui_0.3.2_Linux_x86_64.tar.gz`

You can follow these steps for other Operating Systems as well. Note that for Apple computers like Macs, the Operating System is **Darwin**.

## GO Toolchain
Ensure that the **go** binary is present in your system before following the listed steps.

### Install
To install it directly from the repository without having to compile anything, use:
```
go install github.com/darkhz/invidtui@latest
``` 
	
Note that the installed binary may be present in ~/go/bin, so ensure that your $PATH points to that directory as well.

### Compile
- Clone the [source](https://github.com/darkhz/invidtui/) into a folder using git, like so:<br/><br/>
   ```
   git clone https://github.com/darkhz/invidtui/
   ```
  The source should be cloned into a directory named "**invidtui**".<br/><br/>
  
- Next, change the directory to the "**invidtui**" folder:<br/><br/>
  ```
  cd invidtui
  ```
	
- Finally, use the go toolchain to build the project:<br/><br/>
  ```
  go build main.go -o invidtui-bin
  ```
	
  After the build process, a binary named "**invidtui-bin**" should be present in your current directory. 

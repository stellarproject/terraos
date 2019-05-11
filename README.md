# Terra OS

![terra](iso/splash.png)

Modern, minimal operating system (we've heard that before) optimized for containers within the Stellar Project.


## Build your own OS

Terra is a toolkit for building your own OS for your servers.
Each server can be created as an image that can be run via Docker or containerd to test the install and functionality.

To create a new server image, create a toml file with an id and any other information needed.

```toml
id = "example"
version = "v1"
os = "docker.io/stellarproject/terraos:v9"
repo = "docker.io/stellarproject"
userland = "RUN echo 'terra:terra' | chpasswd"
```

After you have your toml file created with the settings that you want for your OS just run:

```bash
> terra create <my.toml>
```

If you want to push your resulting image to a registry then use the `--push` flag on the create command.

## Global Images

Terra OS publishes two global images for use.

* `stellarproject/kernel:<version>` - Kernel build for terra
* `stellarproject/terraos:<version>` - Terra userland of the distro

## License

```
Copyright (c) 2019 Stellar Project

Permission is hereby granted, free of charge, to any person
obtaining a copy of this software and associated documentation
files (the "Software"), to deal in the Software without
restriction, including without limitation the rights to use, copy,
modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED,
INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
HOLDERS BE LIABLE FOR ANY CLAIM,
DAMAGES OR OTHER LIABILITY,
WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH
THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

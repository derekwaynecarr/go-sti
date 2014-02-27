package sti

import (
	"github.com/fsouza/go-dockerclient"
	"log"
)

// Configuration specifies basic configuration parameters of the library.
type Configuration struct {
	DockerSocket  string
	DockerTimeout int
	WorkingDir    string
	Debug         bool
}

// TODO: collapse with Request?

// Request contains essential fields for any request: a Configuration, a base image, and an
// optional runtime image.
type Request struct {
	Configuration
	BaseImage    string
	RuntimeImage string
}

// requestHandler encapsulates dependencies needed to fulfill requests.
type requestHandler struct {
	dockerClient *docker.Client
	debug        bool
}

func newHandler(req *Request) (*requestHandler, error) {
	dockerClient, err := docker.NewClient(req.DockerSocket)

	if err != nil {
		return nil, ErrDockerConnectionFailed
	}

	return &requestHandler{dockerClient, req.Debug}, nil
}

func (c requestHandler) isImageInLocalRegistry(imageName string) (bool, error) {
	image, err := c.dockerClient.InspectImage(imageName)

	if image != nil {
		return true, nil
	} else if err == docker.ErrNoSuchImage {
		return false, nil
	}

	return false, err
}

func (c requestHandler) containerFromImage(imageName string) (*docker.Container, error) {
	config := docker.Config{Image: imageName, AttachStdout: false, AttachStderr: false, Cmd: []string{"/bin/true"}}
	container, err := c.dockerClient.CreateContainer(docker.CreateContainerOptions{Name: "", Config: &config})
	if err != nil {
		return nil, err
	}

	err = c.dockerClient.StartContainer(container.ID, &docker.HostConfig{})
	if err != nil {
		return nil, err
	}

	exitCode, err := c.dockerClient.WaitContainer(container.ID)
	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		log.Printf("Container exit code: %d\n", exitCode)
		return nil, ErrCreateContainerFailed
	}

	return container, nil
}

func (c requestHandler) checkAndPull(imageName string) (*docker.Image, error) {
	image, err := c.dockerClient.InspectImage(imageName)
	if err != nil {
		return nil, ErrPullImageFailed
	}

	if image == nil {
		if c.debug {
			log.Printf("Pulling image %s\n", imageName)
		}

		err = c.dockerClient.PullImage(docker.PullImageOptions{Repository: imageName}, docker.AuthConfiguration{})
		if err != nil {
			return nil, ErrPullImageFailed
		}

		image, err = c.dockerClient.InspectImage(imageName)
		if err != nil {
			return nil, err
		}
	} else if c.debug {
		log.Printf("Image %s available locally\n", imageName)
	}

	return image, nil
}

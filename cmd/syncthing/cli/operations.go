package cli

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/fs"
)

type folderOverrideCommand struct {
	FolderID string `arg:""`
}

type defaultIgnoresCommand struct {
	Path string `arg:""`
}

type operationCommand struct {
	Restart        struct{}              `cmd:"" help:"Restart syncthing"`
	Shutdown       struct{}              `cmd:"" help:"Shutdown syncthing"`
	FolderOverride folderOverrideCommand `cmd:"" help:"Override changes on folder (remote for sendonly, local for receiveonly). WARNING: Destructive - deletes/changes your data"`
	DefaultIgnores defaultIgnoresCommand `cmd:"" help:"Set the default ignores (config) from a file"`
}

func (*operationCommand) Run(ctx Context, kongCtx *kong.Context) error {
	f := ctx.clientFactory

	switch kongCtx.Selected().Name {
	case "restart":
		return emptyPost("system/restart", f)
	case "shutdown":
		return emptyPost("system/shutdown", f)
	}
	return nil
}

func (f *folderOverrideCommand) Run(ctx Context) error {
	client, err := ctx.clientFactory.getClient()
	if err != nil {
		return err
	}
	cfg, err := getConfig(client)
	if err != nil {
		return err
	}
	rid := f.FolderID
	for _, folder := range cfg.Folders {
		if folder.ID == rid {
			response, err := client.Post("db/override", "")
			if err != nil {
				return err
			}
			if response.StatusCode != 200 {
				errStr := fmt.Sprint("Failed to override changes\nStatus code: ", response.StatusCode)
				bytes, err := responseToBArray(response)
				if err != nil {
					return err
				}
				body := string(bytes)
				if body != "" {
					errStr += "\nBody: " + body
				}
				return errors.New(errStr)
			}
			return nil
		}
	}
	return fmt.Errorf("Folder %q not found", rid)
}

func (d *defaultIgnoresCommand) Run(ctx Context) error {
	client, err := ctx.clientFactory.getClient()
	if err != nil {
		return err
	}
	dir, file := filepath.Split(d.Path)
	filesystem := fs.NewFilesystem(fs.FilesystemTypeBasic, dir)

	fd, err := filesystem.Open(file)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(fd)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	fd.Close()
	if err := scanner.Err(); err != nil {
		return err
	}

	_, err = client.PutJSON("config/defaults/ignores", config.Ignores{Lines: lines})
	return err
}

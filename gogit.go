package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-ini/ini"
)

type gitRepository struct {
	worktree, gitdir string
	conf             *ini.File
}

func newGitRepository(path string, force bool) *gitRepository {
	repo := new(gitRepository)
	repo.worktree = path
	repo.gitdir = filepath.Join(path, ".git")

	if stat, _ := os.Stat(repo.gitdir); !force && !stat.IsDir() {
		panic(errors.New("not a git repository"))
	}

	configPath, err := obtainFile(repo, false, "config")

	if !force && err != nil {
		panic(errors.New("configuration file missing"))
	} else if _, err := os.Stat(configPath); err == nil {
		repo.conf, err = ini.Load(configPath)
		if err != nil {
			panic(errors.New("configuration file read error"))
		}
	}

	if !force {
		ver, err := repo.conf.Section("conf").Key("repositoryformatversion").Int()
		if ver != 0 || err != nil {
			panic(errors.New("unsupported repositoryformatversion"))
		}
	}

	return repo
}

// credit for isDirEmpty: https://stackoverflow.com/a/30708914/8946910
func isDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func getPath(repo *gitRepository, paths ...string) string {
	var newPath string
	for _, path := range paths {
		newPath = filepath.Join(newPath, path)
	}
	return filepath.Join(repo.gitdir, newPath)
}

func makePath(paths ...string) string {
	var newPath string
	for _, path := range paths {
		newPath = filepath.Join(newPath, path)
	}
	return newPath
}

func obtainFile(repo *gitRepository, mkdir bool, paths ...string) (string, error) {
	if _, err := obtainDir(repo, mkdir, paths[:len(paths)-1]...); err == nil {
		return getPath(repo, paths...), nil
	}
	return "", errors.New("not a file")
}

func obtainDir(repo *gitRepository, mkdir bool, paths ...string) (string, error) {
	path := getPath(repo, paths...)

	if stat, err := os.Stat(path); err == nil {
		if stat.IsDir() {
			return path, nil
		}
		return "", errors.New(path + " is not a directory!")
	}

	if mkdir {
		os.MkdirAll(path, os.ModePerm)
		return path, nil
	}
	return "", errors.New(path + " does not exist! Supply mkdir as true to create.")
}

func gitInit(path string) {
	repo := newGitRepository(path, true)

	if stat, err := os.Stat(repo.worktree); !os.IsNotExist(err) {
		if !stat.IsDir() {
			panic(errors.New(path + " is not a directory"))
		}
		if is, err := isDirEmpty(repo.worktree); !is || err != nil {
			panic(errors.New(path + " is not empty"))
		}

	} else {
		obtainDir(repo, true, path)
	}

	obtainDir(repo, true, "branches")
	obtainDir(repo, true, "objects")
	obtainDir(repo, true, "refs", "tags")
	obtainDir(repo, true, "refs", "heads")

	// TODO: Clean code
	fPath, _ := obtainFile(repo, false, "description")

	f, err := os.Create(fPath)
	if err != nil {
		panic(err)
	}
	f.WriteString("Unnamed repository; edit this file 'description' to name the repository.\n")
	f.Close()

	fPath, _ = obtainFile(repo, false, "HEAD")
	f, err = os.Create(fPath)
	if err != nil {
		panic(err)
	}
	f.WriteString("ref: refs/heads/master\n")
	f.Close()

	fPath, _ = obtainFile(repo, false, "config")
	f, err = os.Create(fPath)
	if err != nil {
		panic(err)
	}
	conf, err := ini.Load(fPath)
	if err != nil {
		panic(err)
	}
	conf.Section("core").Key("repositoryformatversion").SetValue("0")
	conf.Section("core").Key("filemode").SetValue("false")
	conf.Section("core").Key("bare").SetValue("false")
	conf.SaveTo(fPath)
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Please supply the command")
		return
	}
	pwd, _ := filepath.Abs("./a")
	switch os.Args[1] {
	case "init":
		gitInit(pwd) // TODO: support additional directories
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}
}

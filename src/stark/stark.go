package main

import (
	"os"
	"path/filepath"
	"strings"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"model"
	"github.com/pborman/uuid"
)

type Tony interface {
	Build()
	CheckHome()
	CheckProjectFile()
	Clean()
	Deploy2Nexus()
	Dist()
	GetProject()
	InstallGB()
	Release()
	RunTests()
}

type Stark struct{}

func (s Stark) Build() {
	os.Chdir(*STARK_PROJECT_PATH)
	logger.Debug("Building go project @", *STARK_PROJECT_PATH)
	rt.RunCommandLogStream("gb", []string{"build"})

}

func (s Stark) Clean() {
	dir, err := filepath.Abs(filepath.Dir(*STARK_PROJECT_PATH))
	os.Chdir(dir)
	if err != nil {
		logger.Fatal(err)
	}
	resp := wiz.Question("You are about to clean all binaries and packages from " + dir + "\nProceed: [y/N] ")
	if strings.ToLower(resp) == "y" {
		os.Remove(dir + "/dist")
		os.Remove(dir + "/pkg")
		os.Remove(dir + "/bin")
	} else {
		fmt.Println("Aborted")
	}
}

func (s Stark) CheckHome() {

	if _, err := os.Stat(STARK_HOME); os.IsNotExist(err) {
		os.Mkdir(STARK_HOME, 00750)
	}
}

//func (s Stark) CheckProjectFile() {
//	if _, err := os.Stat(STARK_FILE); os.IsNotExist(err) {
//		ct.Foreground(ct.Red, false)
//		resp := wiz.Question("Project doesn't have stark.json file. Would you like to create one? [y/N] ")
//		project := model.Project{}
//		project.Version = "0.0.1-SNAPSHOT"
//		if strings.ToLower(resp) == "y" {
//			slice := strings.Split(dir, "/")
//			projectName := slice[len(slice) - 1]
//			ct.Foreground(ct.Cyan, false)
//			pName := wiz.QuestionF("Project Name: [%s] ", projectName)
//			if pName == "" {
//				if pName == "" {
//					project.Name = projectName
//				} else {
//					project.Name = pName
//				}
//			}
//
//			pGroup := wiz.QuestionF("GroupId: [%s] ", "com.company.my")
//			if pGroup == "" {
//				project.GroupId = "com.company.my"
//			} else {
//				project.GroupId = pGroup
//			}
//
//			pArti := wiz.QuestionF("ArtifactId: [%s] ", projectName)
//			if pArti == "" {
//				project.ArtifactId = projectName
//			} else {
//				project.ArtifactId = pArti
//			}
//
//			packaging := wiz.QuestionF("Packaging: [%s] ", "tar.gz")
//			if pArti == "" {
//				project.Packaging = "tar.gz"
//			} else {
//				project.Packaging = packaging
//			}
//
//			file, _ := json.MarshalIndent(&project, "", "  ")
//			wr := []byte(file)
//
//			err := ioutil.WriteFile(dir + "/stark.json", wr, 0750)
//			if err != nil {
//				logger.Fatal(err)
//			}
//			ct.Foreground(ct.Yellow, false)
//			fmt.Println("Writing gobuilder config file...")
//			fmt.Println(string(wr))
//			ct.Foreground(ct.White, false)
//		} else {
//			fmt.Println("Aborted")
//			ct.Foreground(ct.White, false)
//			os.Exit(1)
//		}
//	}
//}

func (s Stark) Dist() {
	if (project.ProjectType == "go") {
		dir, err := filepath.Abs(filepath.Dir(*STARK_PROJECT_PATH))
		os.Chdir(dir)
		if err != nil {
			logger.Fatal(err)
		}
		s.Clean()
		os.Setenv("GOOS", "darwin")
		os.Setenv("GOARCH", "amd64")
		s.Build()
		os.Setenv("GOOS", "linux")
		os.Setenv("GOARCH", "amd64")
		s.Build()
		os.Setenv("GOOS", "windows")
		os.Setenv("GOARCH", "amd64")
		s.Build()
		compressor.Tar(dir + "/dist", "dist.tar.gz")
	} else if project.ProjectType == "static" {
		fmt.Println("Packaging Static Project")
		tmpDir := "/tmp/" + uuid.New()

		fmt.Println(os.Getwd())
		source, _ := os.Getwd()

		fileUtils.CopyDir(source, tmpDir + "/" + project.ArtifactId)

		os.RemoveAll("dist")
		if e, _ := fileUtils.Exists("dist"); !e {
			os.Mkdir("dist", 00750)
		}

		distName := fmt.Sprintf(source + "/dist/%s-%s.%s", project.ArtifactId, project.Version, project.Packaging)
		os.Chdir(tmpDir)
		fmt.Println(tmpDir)
		fmt.Println(distName)
		compressor.Tar(project.ArtifactId, distName)
		os.RemoveAll(tmpDir)
	}
}

func (s Stark) InstallGB() {
	workdir := "/tmp/" + uuid.New()
	os.Chdir(workdir)
	os.Setenv("GOPATH", workdir)
	rt.RunCommandLogStream("go", []string{"get", "github.com/constabulary/gb/..."})
	fmt.Println("Copying GB binaries to /bin/")
	rt.RunCommandLogStream("sudo", []string{"cp", workdir + "/bin/gb", "/bin/gb"})
	rt.RunCommandLogStream("sudo", []string{"cp", workdir + "/bin/gb-vendor", "/bin/gbv-endor"})
	os.RemoveAll(workdir)
	fmt.Println("Done")
}

func (s Stark) RunTests() {
	dir, err := filepath.Abs(filepath.Dir(*STARK_PROJECT_PATH))
	os.Chdir(dir)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Debug("Runing tests @", dir)
	rt.RunCommandLogStream("gb", []string{"test"})
}

func (s Stark) Deploy2Nexus() {
	if !rt.CheckBinaryInPath("mvn") {
		logger.Fatal("Maven not found in PATH, please check your configuration.")
	} else {
		project := s.GetProject()
		var jobRepo model.Repository
		for _, rp := range project.Repositories {
			if rp.Id == *deployToNexusRepId {
				jobRepo = rp
			}
		}

		args := []string{
			"deploy:deploy-file",
			"-DgroupId=" + project.GroupId,
			"-DartifactId=" + project.ArtifactId,
			"-Dversion=" + project.Version,
			"-Dpackaging=" + project.Packaging,
			"-Durl=" + jobRepo.URL,
			"-Dfile=" + *deployToNexusFile,
			"-DgeneratePom=" + *deployToNexusGeneratePom,
			"-DrepositoryId=" + jobRepo.Id,
		}
		err := rt.RunCommandLogStream("mvn", args)
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func (s Stark) GetProject() (model.Project) {
	projectFile, err := ioutil.ReadFile(STARK_FILE)
	var project model.Project
	if err != nil {
		logger.Error("Workspace busted")
	} else {
		var c model.Project
		err := json.Unmarshal(projectFile, &c)
		if err != nil {
			logger.Error(err)
		}
		project = c
	}

	return project
}

func (s Stark) Release() {
	fmt.Println("Releasing project", app.Name, "-", STARK_VERSION)
}
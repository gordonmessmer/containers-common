package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Connections conf", func() {
	var (
		connectionsConfFile string
		containersConfFile  string
	)

	BeforeEach(func() {
		dir := GinkgoT().TempDir()
		connectionsConfFile = filepath.Join(dir, "connections.conf")
		err := os.Setenv("PODMAN_CONNECTIONS_CONF", connectionsConfFile)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		containersConfFile = filepath.Join(dir, "containers.conf")
		err = os.Setenv(containersConfEnv, containersConfFile)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	AfterEach(func() {
		err := os.Unsetenv("PODMAN_CONNECTIONS_CONF")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		err = os.Unsetenv(containersConfEnv)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	It("read non existent file", func() {
		conf, path, err := readConnectionConf()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(path).To(gomega.Equal(connectionsConfFile))
		gomega.Expect(conf).To(gomega.Equal(&ConnectionsFile{}))
	})

	It("read empty file", func() {
		err := os.WriteFile(connectionsConfFile, []byte("{}"), os.ModePerm)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		conf, path, err := readConnectionConf()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(path).To(gomega.Equal(connectionsConfFile))
		gomega.Expect(conf).To(gomega.Equal(&ConnectionsFile{}))
	})

	It("write and read file", func() {
		orgConf := &ConnectionsFile{Connection: ConnectionConfig{
			Default: "test",
			Connections: map[string]Destination{
				"test": {URI: "ssh://podman.io"},
			},
		}, Farm: FarmConfig{
			Default: "farm1",
			List: map[string][]string{
				"farm1": {"test"},
			},
		}}
		err := writeConnectionConf(connectionsConfFile, orgConf)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		conf, path, err := readConnectionConf()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(path).To(gomega.Equal(connectionsConfFile))
		gomega.Expect(conf).To(gomega.Equal(orgConf))
	})

	Context("GetConnection/Farm", func() {
		const defConnectionsConf = `{"Connection":{"Default":"test","Connections":{"test":{"URI":"ssh://podman.io"},"QA":{"URI":"ssh://test","Identity":".ssh/id","IsMachine":true}}},"farm":{"Default":"farm1","List":{"farm1":["test"]}}}`
		const defContainersConf = `
[engine]
  active_service = "containers"
  [engine.service_destinations]
    [engine.service_destinations.containers]
      uri = "unix:///tmp/test.sock"
	  is_machine = true

[farms]
  default = "farm2"
  [farms.list]
    farm2 = ["containers"]
`

		BeforeEach(func() {
			err := os.WriteFile(connectionsConfFile, []byte(defConnectionsConf), os.ModePerm)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			err = os.WriteFile(containersConfFile, []byte(defContainersConf), os.ModePerm)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		It("GetConnection()", func() {
			conf, err := New(nil)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			con, err := conf.GetConnection("", true)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(con).To(gomega.Equal(&Connection{
				Name:        "test",
				Default:     true,
				ReadWrite:   true,
				Destination: Destination{URI: "ssh://podman.io"},
			}))

			con, err = conf.GetConnection("test", false)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(con).To(gomega.Equal(&Connection{
				Name:        "test",
				Default:     true,
				ReadWrite:   true,
				Destination: Destination{URI: "ssh://podman.io"},
			}))

			con, err = conf.GetConnection("QA", false)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(con).To(gomega.Equal(&Connection{
				Name:        "QA",
				Default:     false,
				ReadWrite:   true,
				Destination: Destination{URI: "ssh://test", Identity: ".ssh/id", IsMachine: true},
			}))

			con, err = conf.GetConnection("containers", false)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(con).To(gomega.Equal(&Connection{
				Name:        "containers",
				Default:     false,
				ReadWrite:   false,
				Destination: Destination{URI: "unix:///tmp/test.sock", IsMachine: true},
			}))
		})

		It("GetAllConnections()", func() {
			conf, err := New(nil)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			cons, err := conf.GetAllConnections()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(cons).To(gomega.ContainElements(
				Connection{
					Name:        "test",
					Default:     true,
					ReadWrite:   true,
					Destination: Destination{URI: "ssh://podman.io"},
				},
				Connection{
					Name:        "QA",
					Default:     false,
					ReadWrite:   true,
					Destination: Destination{URI: "ssh://test", Identity: ".ssh/id", IsMachine: true},
				},
				Connection{
					Name:        "containers",
					Default:     false,
					ReadWrite:   false,
					Destination: Destination{URI: "unix:///tmp/test.sock", IsMachine: true},
				},
			))
		})

		It("GetFarmConnections()", func() {
			conf, err := New(nil)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			name, cons, err := conf.GetDefaultFarmConnections()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(name).To(gomega.Equal("farm1"))
			gomega.Expect(cons).To(gomega.ContainElements(
				Connection{
					Name:        "test",
					Default:     false,
					ReadWrite:   false,
					Destination: Destination{URI: "ssh://podman.io"},
				},
			))

			cons, err = conf.GetFarmConnections("farm1")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(cons).To(gomega.ContainElements(
				Connection{
					Name:        "test",
					Default:     false,
					ReadWrite:   false,
					Destination: Destination{URI: "ssh://podman.io"},
				},
			))

			cons, err = conf.GetFarmConnections("farm2")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(cons).To(gomega.ContainElements(
				Connection{
					Name:        "containers",
					Default:     false,
					ReadWrite:   false,
					Destination: Destination{URI: "unix:///tmp/test.sock", IsMachine: true},
				},
			))
		})

		It("GetAllFarms()", func() {
			conf, err := New(nil)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			farms, err := conf.GetAllFarms()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(farms).To(gomega.ContainElements(
				Farm{
					Name:        "farm1",
					Connections: []string{"test"},
					Default:     true,
					ReadWrite:   true,
				},
				Farm{
					Name:        "farm2",
					Connections: []string{"containers"},
					Default:     false,
					ReadWrite:   false,
				},
			))
		})
	})
})

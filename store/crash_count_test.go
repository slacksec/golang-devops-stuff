package store_test

import (
	"code.cloudfoundry.org/workpool"
	. "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/storeadapter/storenodematchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
)

var _ = Describe("Crash Count", func() {
	var (
		store        Store
		storeAdapter storeadapter.StoreAdapter
		conf         *config.Config
		crashCount1  models.CrashCount
		crashCount2  models.CrashCount
		crashCount3  models.CrashCount
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		Expect(err).NotTo(HaveOccurred())
		wpool, err := workpool.NewWorkPool(conf.StoreMaxConcurrentRequests)
		Expect(err).NotTo(HaveOccurred())
		storeAdapter, err = etcdstoreadapter.New(
			&etcdstoreadapter.ETCDOptions{ClusterUrls: etcdRunner.NodeURLS()},
			wpool,
		)
		Expect(err).NotTo(HaveOccurred())
		err = storeAdapter.Connect()
		Expect(err).NotTo(HaveOccurred())

		crashCount1 = models.CrashCount{AppGuid: models.Guid(), AppVersion: models.Guid(), InstanceIndex: 1, CrashCount: 17}
		crashCount2 = models.CrashCount{AppGuid: models.Guid(), AppVersion: models.Guid(), InstanceIndex: 4, CrashCount: 17}
		crashCount3 = models.CrashCount{AppGuid: models.Guid(), AppVersion: models.Guid(), InstanceIndex: 3, CrashCount: 17}

		store = NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
	})

	AfterEach(func() {
		storeAdapter.Disconnect()
	})

	Describe("Saving crash state", func() {
		BeforeEach(func() {
			err := store.SaveCrashCounts(crashCount1, crashCount2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("stores the passed in crash state", func() {
			expectedTTL := uint64(conf.MaximumBackoffDelay().Seconds()) * 2

			node, err := storeAdapter.Get("/hm/v1/apps/crashes/" + crashCount1.AppGuid + "," + crashCount1.AppVersion + "/1")
			Expect(err).NotTo(HaveOccurred())
			Expect(node).To(storenodematchers.MatchStoreNode(storeadapter.StoreNode{
				Key:   "/hm/v1/apps/crashes/" + crashCount1.AppGuid + "," + crashCount1.AppVersion + "/1",
				Value: crashCount1.ToJSON(),
				TTL:   expectedTTL,
			}))

			node, err = storeAdapter.Get("/hm/v1/apps/crashes/" + crashCount2.AppGuid + "," + crashCount2.AppVersion + "/4")
			Expect(err).NotTo(HaveOccurred())
			Expect(node).To(storenodematchers.MatchStoreNode(storeadapter.StoreNode{
				Key:   "/hm/v1/apps/crashes/" + crashCount2.AppGuid + "," + crashCount2.AppVersion + "/4",
				Value: crashCount2.ToJSON(),
				TTL:   expectedTTL,
			}))
		})
	})
})

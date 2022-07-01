package schema

const (
	ConstTx = `
			{
				"format": 2,
				"id": "e8NNxYmPRVgESMA6cu31OAIe-3wvpWbu9Ng3FhK4FbU",
				"last_tx": "GuZGTekm3ZNQWf_I7h-HKeMNxyU8lnC9XZj13zzOv3GIj8TruJARxYcIYuCMgrBi",
				"owner": "xEeBu1pp2HX90_eKDR5BCjNfMVVp-XHBSDp2hDZ_fIBsPl1RKSeYXtTR1H9zhO5-rE0CMrYmw_ChFRmN345wGDA3mPkjwQY5QHWhI2EHxvIUE7UmYrCgZ2Hk0NzFUo5nFzzCN0odkG13Y7Uu5FkXwJICZKs7kO0Ur5V36wpzPigx4FJRy1nQRUt4TOk52pfXYN84dPe9ggcuB7vp6SslHrD2NJs2PllWos1Y6EEg-V7PYV3iHtA6jNzV9ySeQAv6jLRJLHFJ8_9COJS9emu9HGbry9ppLb_78ijb7THygyTPf2kIkoBYLCC7BQwCJFS72nm60lK5ZnZgDJLWSXFk6O1QozK7EuLRxMwk792jTSaRGnRUa-0Wf5SJdnVFKX_GRkmifTvHOof3ipsTgz_Vpdrz83EDW04Y7M2uo04ehLlAW81jRA9r81BJr52z1Zgch0BaKRGA6ES-fzT6akXSTPraoUOIfugWwBfPiHjhcbupWTrfk6ai6vsBcixh-U78ky7VIW_0m3TU-ar8d03ISXEFPH55jrHdo2rmkJ7zusEyt88ebUA3crmUhfUyoN6r0Y6dEeh-8V8UlEuYuI6oK3gpqdlKKhBt4ngK68bdtuv86GTgC-B2BO0atvWdWYWmmTK3Fz1b66uNscquGbHFLnhFlOVRskjovDqURPIN8QE",
				"tags": [
					{
						"name": "Q29udGVudC1UeXBl",
						"value": "aW1hZ2UvcG5n"
					}
				],
				"target": "",
				"quantity": "0",
				"data": "",
				"data_size": "1078598",
				"data_tree": [],
				"data_root": "CEFixryW0VZosVXdqGA6BLvaCZpJavtQoe0MQ6blFPY",
				"reward": "353184384",
				"signature": "qovhGZtEQko_-zN_cKpRpJZQzg0or2VCDHwXvOqNc8API7Ue_LvDZiRtpyKf_WD3qEjB-knEPxuXn-mGZhHHQpMg3O_puxFdXgDTlx9LJy4oveEpSZF39PE4cPvD3jG_LOAr0oOJzPKUju9SWAZgaYoP-630-u4fKixsyNfv0jDtCOKlLrBt656LW-VsiLRJzjJ3CNOv2QXiIgI7JnbCqg7YNOQYDyWFBSU9ULM_el9TUr8nhXmmDvBPHZf21oZZYt-vFMuCEC_vtrr2LKoG7SQ6qVyhJqGBMhHMpl90vmUApTLgbt3reSi3OnsocgosfZFSGJD3HbaXY-mm2zrjWibUNADva5PKZbVJUf6RGZzt5q04JJsjzG9dwT_bhvZ4YS2X9rVmoYt6zjQ3t_a1QKZokW5G2ass_j4iH0hvlITQ2URQ709jiR14Uq0mQ3xIQNldPVWOO6Hqge6nGtHgAu5aXdREFkZL8brej85OsNvg1NnD5rs9_Lf7EkLxji2SA5S5n4njGBSuXxMTrlkEVT2WSGphKZwHVR2YF8PIjvw1v2-CHcBGEf3x_6-n03eIDETp-nNbCbTGikg7WLcma0EGQPI4si_60Oep0kUDiFLz3oGv2pUvwsfF3T9FWfxkEqXVhgq_JJ-1LC_VQ7g04ZD6hnWk1i0Fn15eYPIgebU"
			}
	`
)

type ArFee struct {
	Base     int64
	PerChunk int64
}

type PeerCount struct {
	Peer  string
	Count int64
}

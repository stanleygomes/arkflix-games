package cheevos_test

import "net/http"

// fakeNoBadgeURL responde achievementsets sem BadgeURL (formato antigo).
type fakeNoBadgeURL struct{}

func (fakeNoBadgeURL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"Success":true,"GameId":1,"Title":"X","ConsoleId":1,"ImageIconUrl":"","Sets":[` +
		`{"AchievementSetId":1,"GameId":1,"Title":null,"Type":"core","ImageIconUrl":"","Leaderboards":[],"Achievements":[` +
		`{"ID":7,"Title":"A","Description":"d","Flags":3,"Points":5,"MemAddr":"0xH1=1","Author":"x","BadgeName":"00042","Created":1,"Modified":1}]}]}`))
}

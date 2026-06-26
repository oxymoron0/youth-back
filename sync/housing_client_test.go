package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchList_OK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("expected User-Agent %s, got %s", userAgent, r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"resultList": [
				{"homeCode": "H001", "homeName": "테스트A", "supplyStatus": "02"},
				{"homeCode": "H002", "homeName": "테스트B", "supplyStatus": "03"}
			]
		}`))
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	items, err := client.FetchList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].HomeCode != "H001" {
		t.Errorf("expected HomeCode H001, got %s", items[0].HomeCode)
	}
}

func TestFetchList_EmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"resultList": []}`))
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	_, err := client.FetchList(context.Background())
	if err == nil {
		t.Fatal("expected error for empty result list")
	}
}

func TestFetchList_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	_, err := client.FetchList(context.Background())
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestFetchList_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	_, err := client.FetchList(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseDetailHTML(t *testing.T) {
	htmlStr := `
		<script>var xpos = "127.0509544"; var ypos = "37.5806024";</script>
		<p>대표전화 : 02-6213-5150</p>
		<p>최초모집공고일 : 2026.06.16</p>
		<p>입주(예정)일 : 2026-07-20</p>
		<p>규모 : 총 206 세대 (공공임대 100 세대)</p>
		<p>시행사 : (주)우리대성문</p>
		<p>시공사 : ㈜대성문</p>
		<p>홈페이지 : <a href="https://www.queensw.com">바로가기</a></p>
	`
	f := parseDetailHTML(htmlStr)

	if f.Longitude == nil || f.Latitude == nil {
		t.Fatalf("expected coordinates, got lon=%v lat=%v", f.Longitude, f.Latitude)
	}
	if *f.Longitude != 127.0509544 || *f.Latitude != 37.5806024 {
		t.Errorf("coords mismatch: lon=%v lat=%v", *f.Longitude, *f.Latitude)
	}
	if f.Phone != "02-6213-5150" {
		t.Errorf("phone: %q", f.Phone)
	}
	if f.FirstRecruitDate != "2026-06-16" { // normalized from 2026.06.16
		t.Errorf("first_recruit: %q", f.FirstRecruitDate)
	}
	if f.MoveInDate != "2026-07-20" {
		t.Errorf("move_in: %q", f.MoveInDate)
	}
	if f.TotalUnits != "총 206 세대 (공공임대 100 세대)" {
		t.Errorf("total_units: %q", f.TotalUnits)
	}
	if f.Developer != "(주)우리대성문" {
		t.Errorf("developer: %q", f.Developer)
	}
	if f.Constructor != "㈜대성문" {
		t.Errorf("constructor: %q", f.Constructor)
	}
	if f.HomepageURL != "https://www.queensw.com" {
		t.Errorf("homepage: %q", f.HomepageURL)
	}
}

func TestParseDetailHTML_OutOfBoundsCoordsIgnored(t *testing.T) {
	// coordinates outside Seoul bounds must be ignored
	f := parseDetailHTML(`<script>var xpos = "0.0"; var ypos = "0.0";</script>`)
	if f.Longitude != nil || f.Latitude != nil {
		t.Errorf("expected nil coords for out-of-bounds, got lon=%v lat=%v", f.Longitude, f.Latitude)
	}
}

func TestParseMinRent(t *testing.T) {
	// Room table: 보증금 월임대료 (예상)관리비 per row; expect the lowest of each.
	htmlStr := `
		<table><thead><tr><th>구분</th><th>공급유형</th><th>신청자격</th>
		<th>전체세대수</th><th>보증금</th><th>월임대료</th><th>(예상)관리비</th></tr></thead>
		<tbody>
		<tr><td>원룸</td><td>25A(20.00)</td><td>청년</td><td>13</td><td>92,000,000</td><td>380,000</td><td>110,156</td></tr>
		<tr><td>원룸</td><td>25A(20.00)</td><td>청년</td><td>42</td><td>110,000,000</td><td>450,000</td><td>110,156</td></tr>
		<tr><td>1.5룸</td><td>51B(41.40)</td><td>신혼부부</td><td>1</td><td>170,000,000</td><td>700,000</td><td>224,118</td></tr>
		</tbody></table>`
	f := parseDetailHTML(htmlStr)
	if f.DepositLow == nil || f.RentalLow == nil {
		t.Fatalf("expected rent, got deposit=%v rental=%v", f.DepositLow, f.RentalLow)
	}
	if *f.DepositLow != 92000000 {
		t.Errorf("deposit_low: got %d, want 92000000", *f.DepositLow)
	}
	if *f.RentalLow != 380000 {
		t.Errorf("rental_low: got %d, want 380000", *f.RentalLow)
	}
}

func TestParseMinRent_NoTable(t *testing.T) {
	// No price table (e.g. closed listing) -> nil rent.
	f := parseDetailHTML(`<p>대표전화 : 02-0000-0000</p>`)
	if f.DepositLow != nil || f.RentalLow != nil {
		t.Errorf("expected nil rent without table, got deposit=%v rental=%v", f.DepositLow, f.RentalLow)
	}
}

func TestFetchList_MoneyFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// API returns mixed types for money fields (int or string)
		w.Write([]byte(`{
			"resultList": [
				{"homeCode": "H001", "homeName": "A", "supplyStatus": "02", "moneyDepositLow": 5000, "moneyRentalLow": "200"}
			]
		}`))
	}))
	defer ts.Close()

	client := NewHousingClient().WithHTTPClient(ts.Client()).WithListURL(ts.URL)
	items, err := client.FetchList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

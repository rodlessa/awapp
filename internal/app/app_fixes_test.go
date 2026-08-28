package app

import (
	"errors"
	"strings"
	"testing"

	"weatherterm/internal/anim"
	"weatherterm/internal/weather"
)

// The offline picker must explain *why* it is offline (the fetch error
// was previously swallowed — the panel just re-showed the same desc).
func TestOverlayInfoShowsFetchError(t *testing.T) {
	a := &App{mode: modeOffline, lastErr: errors.New("openweathermap: HTTP 401: bad key")}
	info := a.overlayInfo()
	if info.Err == "" || !strings.Contains(info.Err, "401") {
		t.Fatalf("offline panel should surface the fetch error, got Err=%q", info.Err)
	}
	if info.Live {
		t.Fatal("offline mode must not be reported as live")
	}
}

func TestOverlayInfoNoErrorWhenOK(t *testing.T) {
	a := &App{mode: modeLive, lastErr: nil}
	if info := a.overlayInfo(); info.Err != "" {
		t.Fatalf("live mode with no error should have empty Err, got %q", info.Err)
	}
}

// The +/- size keys must clamp to the same 4..60 range the animator uses.
func TestHandleKeySizeClamp(t *testing.T) {
	a := &App{}
	for i := 0; i < 100; i++ {
		a.handleKey('+')
	}
	if a.sizePct != 60 {
		t.Fatalf("sizePct should clamp at 60, got %v", a.sizePct)
	}
	for i := 0; i < 100; i++ {
		a.handleKey('-')
	}
	if a.sizePct != 4 {
		t.Fatalf("sizePct should clamp at 4, got %v", a.sizePct)
	}
}

// reportError must drain-then-send so the consumer sees the latest error.
func TestReportErrorLatestWins(t *testing.T) {
	ch := make(chan error, 1)
	ch <- errors.New("old")
	reportError(ch, errors.New("new"))
	if got := <-ch; got == nil || got.Error() != "new" {
		t.Fatalf("expected the latest error to win, got %v", got)
	}
}

// Rain scenes start with the cloud deck hidden by default; 'o' brings it
// back.
func TestRainStartsWithCloudsHidden(t *testing.T) {
	r := weather.Report{Condition: weather.Rain, Desc: "light rain"}
	an := animatorFor(r, anim.MoonOptions{}, true)
	pr, ok := an.(*anim.Precip)
	if !ok {
		t.Fatalf("rain should produce a Precip, got %T", an)
	}
	if pr.CloudsOn {
		t.Error("rain should start with clouds hidden")
	}
}

// Before the user presses 'o', applyClouds must not override the scene
// default; once toggled, the choice is re-applied to new scenes.
func TestCloudsToggleIsSticky(t *testing.T) {
	a := &App{cloudsOn: true}
	a.animator = anim.NewPrecip(anim.ModeRain, false, false, 5, 270)
	a.animator.(*anim.Precip).SetClouds(false)
	a.applyClouds()
	if a.animator.(*anim.Precip).CloudsOn {
		t.Error("applyClouds before toggle should keep the scene default (hidden)")
	}

	a.toggleClouds() // 'o' -> show clouds
	if !a.animator.(*anim.Precip).CloudsOn {
		t.Error("after 'o', clouds should be shown")
	}

	// A fresh scene after toggling keeps the user's choice.
	a.animator = anim.NewPrecip(anim.ModeRain, false, false, 5, 270)
	a.animator.(*anim.Precip).SetClouds(false)
	a.applyClouds()
	if !a.animator.(*anim.Precip).CloudsOn {
		t.Error("after the user toggled, applyClouds should re-apply their choice (shown)")
	}
}

// Rain at near-freezing temps renders as sleet (rain + snow together);
// warm rain and temperature-less manual picks stay plain rain.
func TestRainSleetByTemperature(t *testing.T) {
	cold := weather.Report{Condition: weather.Rain, Desc: "rain", TempKelvin: 274.15} // ~1°C
	if an := animatorFor(cold, anim.MoonOptions{}, true); an.(*anim.Precip).Mode != anim.ModeSleet {
		t.Error("freezing rain should render as sleet (rain + snow)")
	}
	warm := weather.Report{Condition: weather.Rain, Desc: "rain", TempKelvin: 290.0} // ~17°C
	if an := animatorFor(warm, anim.MoonOptions{}, true); an.(*anim.Precip).Mode != anim.ModeRain {
		t.Error("warm rain should render as plain rain")
	}
	manual := weather.Report{Condition: weather.Rain, Desc: "rain (manual)"}
	if an := animatorFor(manual, anim.MoonOptions{}, true); an.(*anim.Precip).Mode != anim.ModeRain {
		t.Error("manual rain with no temperature should render as plain rain")
	}
}

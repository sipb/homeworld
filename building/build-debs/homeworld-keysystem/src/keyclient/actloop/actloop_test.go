package actloop

import (
	"bytes"
	"errors"
	"log"
	"reflect"
	"testing"
	"time"
)

type FakeAction struct {
	Performed bool
}

func (f *FakeAction) Pending() (bool, error) {
	return !f.Performed, nil
}

func (f *FakeAction) CheckBlocker() error {
	return nil
}

func (f *FakeAction) Info() string {
	return "fakeinfo"
}

func (f *FakeAction) Perform(logger *log.Logger) error {
	if f.Performed {
		return errors.New("Already performed!")
	}
	f.Performed = true
	logger.Print("PERFORMED!")
	return nil
}

func TestNewActLoop(t *testing.T) {
	actions := []Action{&FakeAction{}, &FakeAction{}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	if loop.logger != logger {
		t.Error("wrong logger")
	}
	if !reflect.DeepEqual(loop.actions, actions) {
		t.Error("wrong actions")
	}
	if loop.should_stop {
		t.Error("should not stop")
	}
	if loop.IsCancelled() {
		t.Error("should not have been stopped yet")
	}
	if logbuf.String() != "" {
		t.Error("should not have logged anything")
	}
	if actions[0].(*FakeAction).Performed || actions[1].(*FakeAction).Performed {
		t.Error("should not have been executed")
	}
}

func TestActLoop_Cancel(t *testing.T) {
	actions := []Action{&FakeAction{}, &FakeAction{}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	if loop.IsCancelled() {
		t.Error("should_stop not initialized properly")
	}
	go loop.Cancel()
	loop.Cancel()
	if !loop.IsCancelled() {
		t.Error("should_stop not set properly")
	}
	if logbuf.String() != "" {
		t.Error("should not have logged anything")
	}
}

func TestActLoop_StepSimple(t *testing.T) {
	actions := []Action{&FakeAction{}, &FakeAction{}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	stabilized := loop.Step()
	if stabilized {
		t.Error("should not yet be stabilized")
	}
	if !actions[0].(*FakeAction).Performed {
		t.Error("should have been executed")
	}
	if actions[1].(*FakeAction).Performed {
		t.Error("should not have been executed")
	}
	stabilized = loop.Step()
	if stabilized {
		t.Error("should not yet be stabilized")
	}
	if !actions[0].(*FakeAction).Performed || !actions[1].(*FakeAction).Performed {
		t.Error("should have been executed")
	}
	stabilized = loop.Step()
	if !stabilized {
		t.Error("should be stabilized")
	}
	if logbuf.String() != "[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n" {
		t.Error("should not have logged:", logbuf.String())
	}
}

type FailPendingAction struct {
	ShouldBePending bool
	PerformedAnyway bool
}

func (f *FailPendingAction) Pending() (bool, error) {
	return f.ShouldBePending, errors.New("purposeful failure")
}

func (f *FailPendingAction) CheckBlocker() error {
	return nil
}

func (f *FailPendingAction) Info() string {
	return "fakeinfo"
}

func (f *FailPendingAction) Perform(logger *log.Logger) error {
	if f.PerformedAnyway {
		panic("should not be here")
	}
	f.PerformedAnyway = true
	return nil
}

func TestActLoop_Step_PendingErr_Halt(t *testing.T) {
	actions := []Action{&FailPendingAction{ShouldBePending: false}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	if !loop.Step() {
		t.Error("should not be destabilized")
	}
	if actions[0].(*FailPendingAction).PerformedAnyway {
		t.Error("should not have been performed")
	}
	if logbuf.String() != "[actloop] actloop check error: purposeful failure (in fakeinfo)\n" {
		t.Error("should not have logged:", logbuf.String())
	}
}

func TestActLoop_Step_PendingErr_Cont(t *testing.T) {
	actions := []Action{&FailPendingAction{ShouldBePending: true}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	if loop.Step() {
		t.Error("should not be stabilized")
	}
	if !actions[0].(*FailPendingAction).PerformedAnyway {
		t.Error("should have been performed")
	}
	if logbuf.String() != "[actloop] actloop check error: purposeful failure (in fakeinfo)\n[actloop] action performed: fakeinfo\n" {
		t.Error("should not have logged:", logbuf.String())
	}
}

type BlockedAction struct {
	errortext string
}

func (f *BlockedAction) Pending() (bool, error) {
	return true, nil
}

func (f *BlockedAction) CheckBlocker() error {
	return errors.New(f.errortext)
}

func (f *BlockedAction) Info() string {
	return "fakeinfo"
}

func (f *BlockedAction) Perform(logger *log.Logger) error {
	panic("should not be here")
}

func TestActLoop_Step_BlockedAction_One(t *testing.T) {
	actions := []Action{&BlockedAction{"uncloggable blockage"}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	if loop.Step() {
		t.Error("should not be stabilized")
	}
	if logbuf.String() != "[actloop] ACTLOOP BLOCKED (1)\n[actloop] actloop blocked by: uncloggable blockage\n" {
		t.Error("should not have logged:", logbuf.String())
	}
}

func TestActLoop_Step_BlockedAction_Three(t *testing.T) {
	actions := []Action{&BlockedAction{"uncloggable blockage A"}, &BlockedAction{"uncloggable blockage B"}, &BlockedAction{"uncloggable blockage C"}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "", 0)
	loop := NewActLoop(actions, logger)
	if loop.Step() {
		t.Error("should not be stabilized")
	}
	if logbuf.String() != "ACTLOOP BLOCKED (3)\nactloop blocked by: uncloggable blockage A\nactloop blocked by: uncloggable blockage B\nactloop blocked by: uncloggable blockage C\n" {
		t.Error("should not have logged:", logbuf.String())
	}
}

type FailingAction struct {
	errortext string
}

func (f *FailingAction) Pending() (bool, error) {
	return true, nil
}

func (f *FailingAction) Info() string {
	return "fakeinfo"
}

func (f *FailingAction) CheckBlocker() error {
	return nil
}

func (f *FailingAction) Perform(logger *log.Logger) error {
	return errors.New(f.errortext)
}

// TODO: handling of other actions around failing actions
func TestActLoop_Step_FailingAction_One(t *testing.T) {
	actions := []Action{&FailingAction{"resonance cascade"}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	if !loop.Step() {
		t.Error("should not be destabilized")
	}
	if logbuf.String() != "[actloop] actloop step error: resonance cascade (in fakeinfo)\n" {
		t.Error("should not have logged:", logbuf.String())
	}
}

func TestActLoop_Step_FailingAction_Continuation(t *testing.T) {
	actions := []Action{&FakeAction{}, &FailingAction{"resonance cascade"}, &FakeAction{}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	if loop.Step() {
		t.Error("should not be stabilized")
	}
	if !actions[0].(*FakeAction).Performed {
		t.Error("should have been performed")
	}
	if actions[2].(*FakeAction).Performed {
		t.Error("should not have been performed")
	}
	if logbuf.String() != "[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n" {
		t.Error("should not have logged:", logbuf.String())
	}
	if loop.Step() {
		t.Error("should not be stabilized")
	}
	if !actions[0].(*FakeAction).Performed {
		t.Error("should have been performed")
	}
	if !actions[2].(*FakeAction).Performed {
		t.Error("should have been performed")
	}
	if logbuf.String() != "[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n[actloop] actloop step error: resonance cascade (in fakeinfo)\n[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n" {
		t.Error("should not have logged:", logbuf.String())
	}
}

func TestActLoop_RunSimple(t *testing.T) {
	actions := []Action{&FakeAction{}, &FakeAction{}}
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[actloop] ", 0)
	loop := NewActLoop(actions, logger)
	go func() { time.Sleep(time.Millisecond * 10); loop.Cancel() }()
	loop.Run(time.Nanosecond, time.Nanosecond * 30)
	if !actions[0].(*FakeAction).Performed || !actions[1].(*FakeAction).Performed {
		t.Error("should have been executed")
	}
	if logbuf.String() != "[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n[actloop] ACTLOOP STABILIZED\n" {
		t.Error("should not have logged:", logbuf.String())
	}
}

type TimeAction struct {
	PerformedAt        time.Time
	LastPendingCheckAt time.Time
}

func (t *TimeAction) Pending() (bool, error) {
	t.LastPendingCheckAt = time.Now()
	return t.PerformedAt.IsZero(), nil
}

func (f *TimeAction) Info() string {
	return "timeinfo"
}

func (t *TimeAction) CheckBlocker() error {
	return nil
}

func (t *TimeAction) Perform(logger *log.Logger) error {
	t.PerformedAt = time.Now()
	return nil
}

const TIME_ATTEMPTS = 4

func TestActLoop_CycleTime(t *testing.T) {
	for attempt := 1; attempt <= TIME_ATTEMPTS; attempt++ {
		taction := &TimeAction{}
		actions := []Action{&FakeAction{}, &FakeAction{}, taction}
		logbuf := bytes.NewBuffer(nil)
		logger := log.New(logbuf, "[actloop] ", 0)
		loop := NewActLoop(actions, logger)
		go func() { time.Sleep(time.Millisecond * 100); loop.Cancel() }()
		start := time.Now()
		loop.Run(time.Millisecond * 10, time.Millisecond * 300)
		if !actions[0].(*FakeAction).Performed || !actions[1].(*FakeAction).Performed || taction.PerformedAt.IsZero() {
			t.Error("should have been executed")
		}
		duration := taction.PerformedAt.Sub(start)
		if duration/time.Millisecond < 20 || duration/time.Millisecond >= 25 {
			if attempt < TIME_ATTEMPTS {
				continue // let's try that again...
			} else {
				t.Error("Invalid duration of execution:", duration, "ms")
			}
		}
		if logbuf.String() != "[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n[actloop] action performed: timeinfo\n[actloop] ACTLOOP STABILIZED\n" {
			t.Error("should not have logged:", logbuf.String())
		}
		break
	}
}

func TestActLoop_StableTime(t *testing.T) {
	for attempt := 1; attempt <= TIME_ATTEMPTS; attempt++ {
		taction := &TimeAction{}
		actions := []Action{&FakeAction{}, &FakeAction{}, taction}
		logbuf := bytes.NewBuffer(nil)
		logger := log.New(logbuf, "[actloop] ", 0)
		loop := NewActLoop(actions, logger)
		interstitial := time.Time{}
		go func() {
			time.Sleep(time.Millisecond * 10)
			interstitial = taction.LastPendingCheckAt
			time.Sleep(time.Millisecond * 30)
			loop.Cancel()
		}()
		start := time.Now()
		loop.Run(time.Millisecond, time.Millisecond * 30)
		if !actions[0].(*FakeAction).Performed || !actions[1].(*FakeAction).Performed || taction.PerformedAt.IsZero() {
			t.Error("should have been executed")
		}
		start_latency := interstitial.Sub(start)
		if start_latency/time.Millisecond < 3 || start_latency/time.Millisecond >= 8 {
			if attempt < TIME_ATTEMPTS {
				continue // let's try that again...
			} else {
				t.Errorf("invalid latency for early check: %v", start_latency)
			}
		}
		pause_for := taction.LastPendingCheckAt.Sub(interstitial)
		if pause_for/time.Millisecond < 30 || pause_for/time.Millisecond >= 31 {
			if attempt < TIME_ATTEMPTS {
				continue // let's try that again...
			} else {
				t.Errorf("invalid duration of stabilized pause: %v", pause_for)
			}
		}
		if logbuf.String() != "[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n[actloop] PERFORMED!\n[actloop] action performed: fakeinfo\n[actloop] action performed: timeinfo\n[actloop] ACTLOOP STABILIZED\n" {
			t.Error("should not have logged:", logbuf.String())
		}
		break
	}
}

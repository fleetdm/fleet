"""Unit tests for the calendar sync script (fleet + fleetd)."""

import datetime as dt
import pytest
import sync


# === Regex matching ===

class TestMilestoneRegex:
    def test_fleet_version_matches(self):
        assert sync.VERSION_RE.match("4.93.0")
        assert sync.VERSION_RE.match("4.92.1")

    def test_fleet_version_rejects_fleetd(self):
        assert not sync.VERSION_RE.match("fleetd-v1.62.0")

    def test_fleetd_version_matches(self):
        m = sync.FLEETD_VERSION_RE.match("fleetd-v1.62.0")
        assert m
        assert m.group(1) == "1.62.0"

    def test_fleetd_version_rejects_fleet(self):
        assert not sync.FLEETD_VERSION_RE.match("4.93.0")

    def test_fleetd_version_rejects_rc(self):
        assert not sync.FLEETD_VERSION_RE.match("fleetd-v1.62.0-rc1")

    def test_fleetd_version_rejects_prefix_only(self):
        assert not sync.FLEETD_VERSION_RE.match("fleetd-v")


class TestCalendarEventRegex:
    def test_fleetd_release_day_minor(self):
        m = sync.FLEETD_RELEASE_DAY_RE.match("Release day: fleetd minor release - fleetd-v1.62.0")
        assert m and m.group(1) == "fleetd-v1.62.0"

    def test_fleetd_release_day_patch(self):
        m = sync.FLEETD_RELEASE_DAY_RE.match("Release day: fleetd patch release - fleetd-v1.62.1")
        assert m and m.group(1) == "fleetd-v1.62.1"

    def test_fleetd_rc(self):
        m = sync.FLEETD_RC_RE.match("Release candidate (fleetd next release - fleetd-v1.62.0)")
        assert m and m.group(1) == "fleetd-v1.62.0"

    def test_fleetd_develop(self):
        m = sync.FLEETD_DEVELOP_RE.match("Develop (fleetd next release - fleetd-v1.62.0)")
        assert m and m.group(1) == "fleetd-v1.62.0"

    def test_fleet_events_dont_match_fleetd_regexes(self):
        assert not sync.FLEETD_RELEASE_DAY_RE.match("Release day: minor release - 4.93.0")
        assert not sync.FLEETD_RC_RE.match("Release candidate (next release - 4.93.0)")
        assert not sync.FLEETD_DEVELOP_RE.match("Develop (next release - 4.93.0)")

    def test_fleetd_events_dont_match_fleet_regexes(self):
        assert not sync.RELEASE_DAY_RE.match("Release day: fleetd minor release - fleetd-v1.62.0")
        assert not sync.RC_RE.match("Release candidate (fleetd next release - fleetd-v1.62.0)")
        assert not sync.DEVELOP_RE.match("Develop (fleetd next release - fleetd-v1.62.0)")


# === Title generation ===

class TestTitleGeneration:
    def _fleet_milestone(self, title="4.93.0", due="2026-10-02"):
        return sync.Milestone(number=1, title=title, due=dt.date.fromisoformat(due), product="fleet")

    def _fleetd_milestone(self, title="fleetd-v1.62.0", due="2026-10-02"):
        return sync.Milestone(number=2, title=title, due=dt.date.fromisoformat(due), product="fleetd")

    def test_fleet_release_day_minor(self):
        assert sync.release_day_summary(self._fleet_milestone("4.93.0")) == "Release day: minor release - 4.93.0"

    def test_fleet_release_day_patch(self):
        assert sync.release_day_summary(self._fleet_milestone("4.93.1")) == "Release day: patch release - 4.93.1"

    def test_fleetd_release_day_minor(self):
        assert sync.release_day_summary(self._fleetd_milestone("fleetd-v1.62.0")) == "Release day: fleetd minor release - fleetd-v1.62.0"

    def test_fleetd_release_day_patch(self):
        assert sync.release_day_summary(self._fleetd_milestone("fleetd-v1.62.1")) == "Release day: fleetd patch release - fleetd-v1.62.1"

    def test_fleetd_rc_summary(self):
        assert sync._rc_summary(self._fleetd_milestone()) == "Release candidate (fleetd next release - fleetd-v1.62.0)"

    def test_fleet_rc_summary(self):
        assert sync._rc_summary(self._fleet_milestone()) == "Release candidate (next release - 4.93.0)"

    def test_fleetd_develop_summary(self):
        assert sync._develop_summary(self._fleetd_milestone()) == "Develop (fleetd next release - fleetd-v1.62.0)"

    def test_fleet_develop_summary(self):
        assert sync._develop_summary(self._fleet_milestone()) == "Develop (next release - 4.93.0)"


# === Out-of-band detection ===

class TestOutOfBand:
    def test_normal_cadence_not_oob(self):
        milestones = [
            sync.Milestone(1, "fleetd-v1.61.0", dt.date(2026, 9, 18), product="fleetd"),
            sync.Milestone(2, "fleetd-v1.62.0", dt.date(2026, 10, 2), product="fleetd"),
            sync.Milestone(3, "fleetd-v1.63.0", dt.date(2026, 10, 23), product="fleetd"),
        ]
        sync._mark_out_of_band(milestones)
        assert all(not m.out_of_band for m in milestones)

    def test_squeezed_milestone_is_oob(self):
        milestones = [
            sync.Milestone(1, "fleetd-v1.61.0", dt.date(2026, 9, 18), product="fleetd"),
            sync.Milestone(2, "fleetd-v1.61.1", dt.date(2026, 9, 25), product="fleetd"),  # 7 days gap both sides
            sync.Milestone(3, "fleetd-v1.62.0", dt.date(2026, 10, 2), product="fleetd"),
        ]
        sync._mark_out_of_band(milestones)
        assert not milestones[0].out_of_band
        assert milestones[1].out_of_band
        assert not milestones[2].out_of_band

    def test_fleet_and_fleetd_independent(self):
        """Fleet and fleetd milestones with same dates should not affect each other's OOB detection."""
        fleet_ms = [
            sync.Milestone(1, "4.92.0", dt.date(2026, 9, 18), product="fleet"),
            sync.Milestone(2, "4.93.0", dt.date(2026, 10, 2), product="fleet"),
        ]
        fleetd_ms = [
            sync.Milestone(3, "fleetd-v1.61.0", dt.date(2026, 9, 18), product="fleetd"),
            sync.Milestone(4, "fleetd-v1.62.0", dt.date(2026, 10, 2), product="fleetd"),
        ]
        sync._mark_out_of_band(fleet_ms)
        sync._mark_out_of_band(fleetd_ms)
        assert all(not m.out_of_band for m in fleet_ms + fleetd_ms)


# === Categorize ===

class TestCategorize:
    def _event(self, summary):
        return sync.CalEvent(id="x", summary=summary, start=dt.date.today(), end=None, raw={})

    def test_fleet_release_day(self):
        cat, ver = sync.categorize(self._event("Release day: minor release - 4.93.0"))
        assert cat == "release_day" and ver == "4.93.0"

    def test_fleetd_release_day(self):
        cat, ver = sync.categorize(self._event("Release day: fleetd minor release - fleetd-v1.62.0"))
        assert cat == "release_day" and ver == "fleetd-v1.62.0"

    def test_fleet_rc(self):
        cat, ver = sync.categorize(self._event("Release candidate (next release - 4.93.0)"))
        assert cat == "rc" and ver == "4.93.0"

    def test_fleetd_rc(self):
        cat, ver = sync.categorize(self._event("Release candidate (fleetd next release - fleetd-v1.62.0)"))
        assert cat == "rc" and ver == "fleetd-v1.62.0"

    def test_unrelated_event(self):
        cat, ver = sync.categorize(self._event("Team standup"))
        assert cat is None and ver is None


# === plan_actions: create events for fleetd ===

class TestPlanActions:
    def test_creates_fleetd_events(self):
        """A fleetd milestone with no existing events should propose 3 creates."""
        milestones = [
            sync.Milestone(1, "fleetd-v1.62.0", dt.date(2026, 10, 2), product="fleetd"),
        ]
        actions = sync.plan_actions(milestones, [], dt.date(2026, 9, 15))
        creates = [a for a in actions if a.kind == "create"]
        summaries = [a.new_summary for a in creates]
        assert "Release day: fleetd minor release - fleetd-v1.62.0" in summaries
        assert "Release candidate (fleetd next release - fleetd-v1.62.0)" in summaries
        assert "Develop (fleetd next release - fleetd-v1.62.0)" in summaries

    def test_creates_fleet_events(self):
        """A fleet milestone with no existing events should still work as before."""
        milestones = [
            sync.Milestone(1, "4.93.0", dt.date(2026, 10, 2), product="fleet"),
        ]
        actions = sync.plan_actions(milestones, [], dt.date(2026, 9, 15))
        creates = [a for a in actions if a.kind == "create"]
        summaries = [a.new_summary for a in creates]
        assert "Release day: minor release - 4.93.0" in summaries
        assert "Release candidate (next release - 4.93.0)" in summaries
        assert "Develop (next release - 4.93.0)" in summaries

    def test_no_cross_matching(self):
        """A fleet event should not match a fleetd milestone even if dates match."""
        milestones = [
            sync.Milestone(1, "fleetd-v1.62.0", dt.date(2026, 10, 2), product="fleetd"),
        ]
        # A fleet release day event on the same date
        fleet_event = sync.CalEvent(
            id="ev1",
            summary="Release day: minor release - 4.93.0",
            start=dt.date(2026, 10, 2),
            end=dt.date(2026, 10, 3),
            raw={"start": {"date": "2026-10-02"}, "end": {"date": "2026-10-03"}},
        )
        actions = sync.plan_actions(milestones, [fleet_event], dt.date(2026, 9, 15))
        # The fleet event should be stale (no matching fleet milestone) and fleetd should get creates
        creates = [a for a in actions if a.kind == "create"]
        fleetd_creates = [a for a in creates if "fleetd" in (a.new_summary or "")]
        assert len(fleetd_creates) == 3  # release day + rc + develop

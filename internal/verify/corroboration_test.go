package verify

import (
	"testing"
)

func TestDefaultCorroborationPolicy(t *testing.T) {
	p := DefaultCorroborationPolicy()
	if p == nil {
		t.Fatal("expected non-nil default policy")
	}
	sum := p.AnchorBonus + p.SignatureBonus + p.SnapshotBonus
	if sum > p.MaxBonus {
		t.Errorf("component bonuses sum %.4f exceeds MaxBonus %.4f", sum, p.MaxBonus)
	}
	if p.MaxBonus <= 0 {
		t.Error("MaxBonus must be positive")
	}
}

func TestCorroborationPolicyValidate(t *testing.T) {
	t.Run("default policy is valid", func(t *testing.T) {
		if err := DefaultCorroborationPolicy().Validate(); err != nil {
			t.Errorf("expected valid default policy, got: %v", err)
		}
	})

	t.Run("nil policy is valid", func(t *testing.T) {
		var p *CorroborationPolicy
		if err := p.Validate(); err != nil {
			t.Errorf("expected nil policy to be valid, got: %v", err)
		}
	})

	t.Run("policy with components summing to MaxBonus is valid", func(t *testing.T) {
		p := &CorroborationPolicy{
			AnchorBonus:    0.05,
			SignatureBonus: 0.03,
			SnapshotBonus:  0.02,
			MaxBonus:       0.10,
		}
		if err := p.Validate(); err != nil {
			t.Errorf("expected valid policy, got: %v", err)
		}
	})

	t.Run("policy with components exceeding MaxBonus is invalid", func(t *testing.T) {
		p := &CorroborationPolicy{
			AnchorBonus:    0.06,
			SignatureBonus: 0.03,
			SnapshotBonus:  0.02,
			MaxBonus:       0.10,
		}
		if err := p.Validate(); err == nil {
			t.Error("expected error for policy where components exceed MaxBonus")
		}
	})
}

func TestComputeCorroborationBonus(t *testing.T) {
	p := DefaultCorroborationPolicy()

	t.Run("nil policy returns zero", func(t *testing.T) {
		if got := computeCorroborationBonus(nil, true, true, true); got != 0.0 {
			t.Errorf("nil policy bonus = %f, want 0.0", got)
		}
	})

	t.Run("no evidence returns zero", func(t *testing.T) {
		if got := computeCorroborationBonus(p, false, false, false); got != 0.0 {
			t.Errorf("no-evidence bonus = %f, want 0.0", got)
		}
	})

	t.Run("anchor only returns AnchorBonus", func(t *testing.T) {
		got := computeCorroborationBonus(p, true, false, false)
		if got != p.AnchorBonus {
			t.Errorf("anchor-only bonus = %f, want %f", got, p.AnchorBonus)
		}
	})

	t.Run("signature only returns SignatureBonus", func(t *testing.T) {
		got := computeCorroborationBonus(p, false, true, false)
		if got != p.SignatureBonus {
			t.Errorf("signature-only bonus = %f, want %f", got, p.SignatureBonus)
		}
	})

	t.Run("snapshot only returns SnapshotBonus", func(t *testing.T) {
		got := computeCorroborationBonus(p, false, false, true)
		if got != p.SnapshotBonus {
			t.Errorf("snapshot-only bonus = %f, want %f", got, p.SnapshotBonus)
		}
	})

	t.Run("anchor and signature returns sum", func(t *testing.T) {
		want := p.AnchorBonus + p.SignatureBonus
		got := computeCorroborationBonus(p, true, true, false)
		if got != want {
			t.Errorf("anchor+sig bonus = %f, want %f", got, want)
		}
	})

	t.Run("all three capped at MaxBonus", func(t *testing.T) {
		got := computeCorroborationBonus(p, true, true, true)
		if got > p.MaxBonus {
			t.Errorf("all-three bonus = %f exceeds MaxBonus %f", got, p.MaxBonus)
		}
		if got != p.MaxBonus {
			t.Errorf("all-three bonus = %f, want MaxBonus %f", got, p.MaxBonus)
		}
	})

	t.Run("cap enforced with higher-than-MaxBonus sum", func(t *testing.T) {
		custom := &CorroborationPolicy{
			AnchorBonus:    0.08,
			SignatureBonus: 0.05,
			SnapshotBonus:  0.05,
			MaxBonus:       0.12,
		}
		got := computeCorroborationBonus(custom, true, true, true)
		if got != custom.MaxBonus {
			t.Errorf("capped bonus = %f, want MaxBonus %f", got, custom.MaxBonus)
		}
	})
}

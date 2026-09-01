package main

import "testing"

func TestResolveAccessProfileAttachmentMutation(t *testing.T) {
	t.Run("non-empty desired list uses desired list", func(t *testing.T) {
		current := []string{"a", "b"}
		desired := []string{"b"}

		got := resolveAccessProfileAttachmentMutation(current, desired)
		if len(got) != 1 || got[0] != "b" {
			t.Fatalf("expected desired list to be used, got %#v", got)
		}
	})

	t.Run("empty desired list detaches current attachments", func(t *testing.T) {
		current := []string{"a", "b", "c"}
		desired := []string{}

		got := resolveAccessProfileAttachmentMutation(current, desired)
		if len(got) != 3 {
			t.Fatalf("expected all current attachments to be detached, got %#v", got)
		}
	})

	t.Run("already empty attachments are treated as no-op", func(t *testing.T) {
		current := []string{}
		desired := []string{}

		got := resolveAccessProfileAttachmentMutation(current, desired)
		if len(got) != 0 {
			t.Fatalf("expected empty mutation to remain empty, got %#v", got)
		}
	})
}

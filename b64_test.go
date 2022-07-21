package main

import (
	"testing"

	"github.com/slack-go/slack"
)

func Test_b64(t *testing.T) {
	type args struct {
		s slack.SlashCommand
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "base case",
			args: args{
				s: slack.SlashCommand{
					Text: "",
				},
			},
			want: "",
		},
		{
			name: "with a string",
			args: args{
				s: slack.SlashCommand{
					Text: "foo",
				},
			},
			want: "Zm9v",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := b64(tt.args.s); got != tt.want {
				t.Errorf("b64() = %v, want %v", got, tt.want)
			}
		})
	}
}

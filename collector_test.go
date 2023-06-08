package storage

import "testing"

func Test_normalizeLink(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "habr",
			args: args{
				in: "https://habr.com/ru/companies/otus/articles/739966/?utm_campaign=16538261\u0026utm_source=vk_flows\u0026utm_medium=social",
			},
			want: "https://habr.com/ru/companies/otus/articles/739966/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLink(tt.args.in); got != tt.want {
				t.Errorf("normalizeLink() = %v, want %v", got, tt.want)
			}
		})
	}
}

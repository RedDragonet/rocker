package image

import (
	"testing"
)

func Test_parseImageUrl(t *testing.T) {
	type args struct {
		imageName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{"nginx", args{"nginx"}, "https://ustc-edu-cn.mirror.aliyuncs.com", "library/nginx", false},
		{"abc/def", args{"abc/def"}, "https://ustc-edu-cn.mirror.aliyuncs.com", "abc/def", false},
		{"http://hub-mirror.c.163.com/nginx", args{"http://hub-mirror.c.163.com/nginx"}, "http://hub-mirror.c.163.com", "library/nginx", false},
		{"http://hub-mirror.c.163.com/abc/def", args{"http://hub-mirror.c.163.com/abc/def"}, "http://hub-mirror.c.163.com", "abc/def", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseImageUrl(tt.args.imageName)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseImageUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseImageUrl() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseImageUrl() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestPullImage(t *testing.T) {
	type args struct {
		imageName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"", args{"nginx"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := PullImage(tt.args.imageName); (err != nil) != tt.wantErr {
				t.Errorf("PullImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

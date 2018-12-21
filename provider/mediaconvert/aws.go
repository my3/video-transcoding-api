// Package mediaconvert provides a implementation of the provider that
// uses AWS Elemental Media Convert for transcoding media files.
//
// It doesn't expose any public type. In order to use the provider, one must
// import this package and then grab the factory from the provider package:
//
//     import (
//         "github.com/NYTimes/video-transcoding-api/provider"
//         "github.com/NYTimes/video-transcoding-api/provider/mediaconvert"
//     )
//
//     func UseProvider() {
//         factory, err := provider.GetProviderFactory(mediaconvert.Name)
//         // handle err and use factory to get an instance of the provider.
//     }
package mediaconvert

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"encoding/json"

	"github.com/NYTimes/video-transcoding-api/config"
	"github.com/NYTimes/video-transcoding-api/db"
	"github.com/NYTimes/video-transcoding-api/provider"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/aws/aws-sdk-go/service/mediaconvert/mediaconvertiface"
)

const (
	// Name is the name used for registering the Elastic Transcoder
	// provider in the registry of providers.
	Name = "mediaconvert"

	defaultAWSRegion = "us-east-1"
	hlsPlayList      = "HLSv3"
)

var (
	errAWSInvalidConfig = errors.New("invalid Media Convert config. Please define the configuration entries in the config file or environment variables")
	s3Pattern           = regexp.MustCompile(`^s3://`)
)

func init() {
	provider.Register(Name, mediaConvertFactory)
}

type awsProvider struct {
	c      mediaconvertiface.MediaConvertAPI
	config *config.MediaConvert
}

func (p *awsProvider) Transcode(job *db.Job) (*provider.JobStatus, error) {

	svc := mediaconvert.New(mySession)

	var params *mediaconvert.CreateJobInput
	err := json.Unmarshal([]byte(sampleJson), params)
	if err != nil {
		return nil, err;
	}

	resp, err := svc.CreateJob(params)
	if err != nil {
		return nil, err;
	}

	return &provider.JobStatus{
		ProviderName:  Name,
		ProviderJobID: aws.StringValue(resp.Job.Id),
		Status:        provider.StatusQueued,
	}, nil
}

func mediaConvertFactory(cfg *config.Config) (provider.TranscodingProvider, error) {
	if cfg.MediaConvert.AccessKeyID == "" || cfg.MediaConvert.SecretAccessKey == "" {
		return nil, errAWSInvalidConfig
	}
	creds := credentials.NewStaticCredentials(cfg.MediaConvert.AccessKeyID, cfg.MediaConvert.SecretAccessKey, "")
	region := cfg.MediaConvert.Region
	if region == "" {
		region = defaultAWSRegion
	}
	awsSession, err := session.NewSession(aws.NewConfig().WithCredentials(creds).WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &awsProvider{
		c:      mediaconvert.New(awsSession),
		config: cfg.MediaConvert,
	}, nil
}

func (p *awsProvider) JobStatus(job *db.Job) (*provider.JobStatus, error) {
}

func (p *awsProvider) CancelJob(id string) error {
	return nil
}

func (p *awsProvider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		InputFormats:  []string{"h264"},
		OutputFormats: []string{"mp4", "hls", "webm"},
		Destinations:  []string{"s3"},
	}
}

func (p *awsProvider) CreatePreset(preset db.Preset) (string, error) {
}

func (p *awsProvider) DeletePreset(presetID string) error {
}

func (p *awsProvider) GetPreset(presetID string) (interface{}, error) {
}

func (p *awsProvider) Healthcheck() error {
}

var sampleJson = `{
    "UserMetadata": {},
    "Role": "ROLE ARN",
    "Settings": {
      "OutputGroups": [
        {
          "Name": "File Group",
          "OutputGroupSettings": {
            "Type": "FILE_GROUP_SETTINGS",
            "FileGroupSettings": {
              "Destination": "s3://bucket/out"
            }
          },
          "Outputs": [
            {
              "VideoDescription": {
                "ScalingBehavior": "DEFAULT",
                "TimecodeInsertion": "DISABLED",
                "AntiAlias": "ENABLED",
                "Sharpness": 50,
                "CodecSettings": {
                  "Codec": "H_264",
                  "H264Settings": {
                    "InterlaceMode": "PROGRESSIVE",
                    "NumberReferenceFrames": 3,
                    "Syntax": "DEFAULT",
                    "Softness": 0,
                    "GopClosedCadence": 1,
                    "GopSize": 48,
                    "Slices": 1,
                    "GopBReference": "DISABLED",
                    "SlowPal": "DISABLED",
                    "SpatialAdaptiveQuantization": "ENABLED",
                    "TemporalAdaptiveQuantization": "ENABLED",
                    "FlickerAdaptiveQuantization": "DISABLED",
                    "EntropyEncoding": "CABAC",
                    "Bitrate": 4500000,
                    "FramerateControl": "SPECIFIED",
                    "RateControlMode": "CBR",
                    "CodecProfile": "HIGH",
                    "Telecine": "NONE",
                    "MinIInterval": 0,
                    "AdaptiveQuantization": "HIGH",
                    "CodecLevel": "LEVEL_4_1",
                    "FieldEncoding": "PAFF",
                    "SceneChangeDetect": "ENABLED",
                    "QualityTuningLevel": "SINGLE_PASS_HQ",
                    "FramerateConversionAlgorithm": "DUPLICATE_DROP",
                    "UnregisteredSeiTimecode": "DISABLED",
                    "GopSizeUnits": "FRAMES",
                    "ParControl": "INITIALIZE_FROM_SOURCE",
                    "NumberBFramesBetweenReferenceFrames": 3,
                    "RepeatPps": "DISABLED",
                    "HrdBufferSize": 9000000,
                    "HrdBufferInitialFillPercentage": 90,
                    "FramerateNumerator": 24000,
                    "FramerateDenominator": 1001
                  }
                },
                "AfdSignaling": "NONE",
                "DropFrameTimecode": "ENABLED",
                "RespondToAfd": "NONE",
                "ColorMetadata": "INSERT",
                "Width": 1920,
                "Height": 1080
              },
              "AudioDescriptions": [
                {
                  "AudioTypeControl": "FOLLOW_INPUT",
                  "CodecSettings": {
                    "Codec": "AAC",
                    "AacSettings": {
                      "AudioDescriptionBroadcasterMix": "NORMAL",
                      "Bitrate": 96000,
                      "RateControlMode": "CBR",
                      "CodecProfile": "LC",
                      "CodingMode": "CODING_MODE_2_0",
                      "RawFormat": "NONE",
                      "SampleRate": 48000,
                      "Specification": "MPEG4"
                    }
                  },
                  "LanguageCodeControl": "FOLLOW_INPUT"
                }
              ],
              "ContainerSettings": {
                "Container": "MP4",
                "Mp4Settings": {
                  "CslgAtom": "INCLUDE",
                  "FreeSpaceBox": "EXCLUDE",
                  "MoovPlacement": "PROGRESSIVE_DOWNLOAD"
                }
              }
            }
          ]
        }
      ],
      "AdAvailOffset": 0,
      "Inputs": [
        {
          "AudioSelectors": {
            "Audio Selector 1": {
              "Tracks": [
                1
              ],
              "Offset": 0,
              "DefaultSelection": "DEFAULT",
              "SelectorType": "TRACK",
              "ProgramSelection": 1
            },
            "Audio Selector 2": {
              "Tracks": [
                2
              ],
              "Offset": 0,
              "DefaultSelection": "NOT_DEFAULT",
              "SelectorType": "TRACK",
              "ProgramSelection": 1
            }
          },
          "VideoSelector": {
            "ColorSpace": "FOLLOW"
          },
          "FilterEnable": "AUTO",
          "PsiControl": "USE_PSI",
          "FilterStrength": 0,
          "DeblockFilter": "DISABLED",
          "DenoiseFilter": "DISABLED",
          "TimecodeSource": "EMBEDDED",
          "FileInput": "s3://input"
        }
      ]
    }
  }`
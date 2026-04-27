package handler

import (
	"time"

	v1 "github.com/raul/monitor/api/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

// timestampToProto конвертирует time.Time в protobuf Timestamp.
func timestampToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// protoToTimestamp конвертирует protobuf Timestamp в time.Time.
func protoToTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

var protoChannelToModelMap = map[v1.TemplateChannel]model.TemplateChannel{
	v1.TemplateChannel_CHANNEL_EMAIL:    model.ChannelEmail,
	v1.TemplateChannel_CHANNEL_TELEGRAM: model.ChannelTelegram,
	v1.TemplateChannel_CHANNEL_WEBHOOK:  model.ChannelWebhook,
	v1.TemplateChannel_CHANNEL_SLACK:    model.ChannelSlack,
	v1.TemplateChannel_CHANNEL_DISCORD:  model.ChannelDiscord,
	v1.TemplateChannel_CHANNEL_SMS:      model.ChannelSMS,
}

var modelChannelToProtoMap = map[model.TemplateChannel]v1.TemplateChannel{
	model.ChannelEmail:    v1.TemplateChannel_CHANNEL_EMAIL,
	model.ChannelTelegram: v1.TemplateChannel_CHANNEL_TELEGRAM,
	model.ChannelWebhook:  v1.TemplateChannel_CHANNEL_WEBHOOK,
	model.ChannelSlack:    v1.TemplateChannel_CHANNEL_SLACK,
	model.ChannelDiscord:  v1.TemplateChannel_CHANNEL_DISCORD,
	model.ChannelSMS:      v1.TemplateChannel_CHANNEL_SMS,
}

var protoTypeToModelMap = map[v1.TemplateType]model.TemplateType{
	v1.TemplateType_TYPE_MONITOR_UP:          model.TypeMonitorUp,
	v1.TemplateType_TYPE_MONITOR_DOWN:        model.TypeMonitorDown,
	v1.TemplateType_TYPE_MONITOR_DEGRADED:    model.TypeMonitorDegraded,
	v1.TemplateType_TYPE_CERTIFICATE_EXPIRY:  model.TypeCertificateExpiry,
	v1.TemplateType_TYPE_FLAPPING_DETECTED:   model.TypeFlappingDetected,
	v1.TemplateType_TYPE_INCIDENT_CREATED:    model.TypeIncidentCreated,
	v1.TemplateType_TYPE_INCIDENT_RESOLVED:   model.TypeIncidentResolved,
	v1.TemplateType_TYPE_MAINTENANCE_STARTED: model.TypeMaintenanceStarted,
	v1.TemplateType_TYPE_MAINTENANCE_ENDED:   model.TypeMaintenanceEnded,
	v1.TemplateType_TYPE_CUSTOM:              model.TypeCustom,
}

var modelTypeToProtoMap = map[model.TemplateType]v1.TemplateType{
	model.TypeMonitorUp:          v1.TemplateType_TYPE_MONITOR_UP,
	model.TypeMonitorDown:        v1.TemplateType_TYPE_MONITOR_DOWN,
	model.TypeMonitorDegraded:    v1.TemplateType_TYPE_MONITOR_DEGRADED,
	model.TypeCertificateExpiry:  v1.TemplateType_TYPE_CERTIFICATE_EXPIRY,
	model.TypeFlappingDetected:   v1.TemplateType_TYPE_FLAPPING_DETECTED,
	model.TypeIncidentCreated:    v1.TemplateType_TYPE_INCIDENT_CREATED,
	model.TypeIncidentResolved:   v1.TemplateType_TYPE_INCIDENT_RESOLVED,
	model.TypeMaintenanceStarted: v1.TemplateType_TYPE_MAINTENANCE_STARTED,
	model.TypeMaintenanceEnded:   v1.TemplateType_TYPE_MAINTENANCE_ENDED,
	model.TypeCustom:             v1.TemplateType_TYPE_CUSTOM,
}

var protoEngineToModelMap = map[v1.TemplateEngine]model.TemplateEngine{
	v1.TemplateEngine_ENGINE_GOTEMPLATE: model.EngineGoTemplate,
	v1.TemplateEngine_ENGINE_JINJA2:     model.EngineJinja2,
	v1.TemplateEngine_ENGINE_HANDLEBARS: model.EngineHandlebars,
}

var modelEngineToProtoMap = map[model.TemplateEngine]v1.TemplateEngine{
	model.EngineGoTemplate: v1.TemplateEngine_ENGINE_GOTEMPLATE,
	model.EngineJinja2:     v1.TemplateEngine_ENGINE_JINJA2,
	model.EngineHandlebars: v1.TemplateEngine_ENGINE_HANDLEBARS,
}

func protoChannelToModel(c v1.TemplateChannel) model.TemplateChannel {
	if m, ok := protoChannelToModelMap[c]; ok {
		return m
	}
	return model.ChannelUnspecified
}

func modelChannelToProto(c model.TemplateChannel) v1.TemplateChannel {
	if p, ok := modelChannelToProtoMap[c]; ok {
		return p
	}
	return v1.TemplateChannel_CHANNEL_UNSPECIFIED
}

func protoTypeToModel(t v1.TemplateType) model.TemplateType {
	if m, ok := protoTypeToModelMap[t]; ok {
		return m
	}
	return model.TemplateType("")
}

func modelTypeToProto(t model.TemplateType) v1.TemplateType {
	if p, ok := modelTypeToProtoMap[t]; ok {
		return p
	}
	return v1.TemplateType_TYPE_UNSPECIFIED
}

func protoEngineToModel(e v1.TemplateEngine) model.TemplateEngine {
	if m, ok := protoEngineToModelMap[e]; ok {
		return m
	}
	return model.TemplateEngine("")
}

func modelEngineToProto(e model.TemplateEngine) v1.TemplateEngine {
	if p, ok := modelEngineToProtoMap[e]; ok {
		return p
	}
	return v1.TemplateEngine_ENGINE_UNSPECIFIED
}

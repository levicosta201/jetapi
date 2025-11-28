package cache

import (
	"github.com/macsencasaus/jetapi/internal/cloudfront"
	"github.com/macsencasaus/jetapi/internal/s3"
)

type Manager struct {
	cache      *Cache
	cloudfront *cloudfront.CloudFront
	s3Manager  *s3.S3Manager
}

func NewManager(cache *Cache, cloudfront *cloudfront.CloudFront, s3Manager *s3.S3Manager) *Manager {
	return &Manager{
		cache:      cache,
		cloudfront: cloudfront,
		s3Manager:  s3Manager,
	}
}

func (m *Manager) GetCache() *Cache {
	return m.cache
}

func (m *Manager) GetCloudFront() *cloudfront.CloudFront {
	return m.cloudfront
}

func (m *Manager) GetS3Manager() *s3.S3Manager {
	return m.s3Manager
}

func (m *Manager) IsCacheAvailable() bool {
	return m.cache != nil && m.cache.IsAvailable()
}

func (m *Manager) IsCloudFrontEnabled() bool {
	return m.cloudfront != nil && m.cloudfront.IsEnabled()
}

func (m *Manager) IsS3Enabled() bool {
	return m.s3Manager != nil && m.s3Manager.IsEnabled()
}


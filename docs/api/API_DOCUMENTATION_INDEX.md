# API Documentation Index

Welcome to the SilentRelay API documentation. This comprehensive guide covers all REST API endpoints and WebSocket protocols for the SilentRelay platform.

## Navigation

### REST API Documentation

- **[Authentication API](API_AUTHENTICATION.md)** - User registration, login, and token management
- **[User Management API](API_USERS.md)** - User profile and key management
- **[Device Management API](API_DEVICES.md)** - Device linking and management
- **[Messaging API](API_MESSAGES.md)** - Message retrieval and status updates
- **[Group API](API_GROUPS.md)** - Group creation and member management
- **[Media API](API_MEDIA.md)** - Media upload and download
- **[Security & Rate Limiting](API_SECURITY.md)** - Error codes, rate limits, and security

### WebSocket API Documentation

- **[WebSocket API](API_WEBSOCKET.md)** - Real-time messaging protocol

### Additional Resources

- **[API Changelog](API_CHANGELOG.md)** - Version history and breaking changes
- **[API Best Practices](API_BEST_PRACTICES.md)** - Recommendations for API consumers

## Quick Start

**Base URL**: `https://api.yourdomain.com/v1`

**Authentication**: All endpoints require JWT authentication via `Authorization: Bearer <token>` header, except for authentication endpoints themselves.

**Content-Type**: `application/json` for all requests and responses.

## API Categories Overview

| Category | Endpoints | Description |
|----------|-----------|-------------|
| **Authentication** | 5 endpoints | User registration, login, token refresh |
| **User Management** | 4 endpoints | Profile management, key retrieval |
| **Device Management** | 3 endpoints | Device linking and primary device management |
| **Device Approval** | 6 endpoints | Secure device linking workflow |
| **Messaging** | 2 endpoints | Message retrieval and status updates |
| **Groups** | 4 endpoints | Group creation and member management |
| **Media** | 6 endpoints | Media upload/download with proxy support |
| **WebSocket** | 1 endpoint | Real-time messaging protocol |

## Version Information

**Current API Version**: v1
**Last Updated**: 2025-12-04
**Documentation Status**: Complete

## Documentation Conventions

- **HTTP Methods**: `GET`, `POST`, `PUT`, `DELETE`
- **Authentication**: Requires auth, Public endpoint
- **Rate Limiting**: Strict rate limits, Normal rate limits
- **Response Codes**: Standard HTTP status codes with detailed error messages

## Getting Started

1. **Register a new user** via `/auth/request-code` and `/auth/verify`
2. **Authenticate** using `/auth/login` to get JWT tokens
3. **Manage devices** using the device approval workflow
4. **Retrieve user keys** for E2EE session establishment
5. **Exchange messages** via WebSocket for real-time communication
6. **Manage media** using presigned URLs for secure uploads/downloads

## API Exploration

Use our interactive API explorer to test endpoints:

- **Swagger UI**: [https://silentrelay.com.au/api/docs](https://silentrelay.com.au/api/docs)
- **Postman Collection**: Available upon request
- **API Sandbox**: Development environment available

## Support & Feedback

For API-related questions or to report documentation issues:

- **Email**: api-support@yourdomain.com
- **GitHub Issues**: [GitHub Repository](https://github.com/yourorg/messaging-app)
- **Developer Community**: [Community Forum](https://community.yourdomain.com)

---
*© 2025 SilentRelay. All rights reserved.*
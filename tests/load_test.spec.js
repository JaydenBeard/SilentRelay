// Load Testing Script for Messaging Application
// Uses Node.js and axios for HTTP requests

const axios = require('axios');
const { performance } = require('perf_hooks');

const BASE_URL = 'http://localhost';
const ENDPOINTS = {
  health: '/health',
  authRequestCode: '/api/v1/auth/request-code',
  messages: '/api/v1/messages'
};

class LoadTester {
  constructor(baseUrl = BASE_URL) {
    this.baseUrl = baseUrl;
    this.results = {
      totalRequests: 0,
      successfulRequests: 0,
      failedRequests: 0,
      responseTimes: [],
      errors: []
    };
  }

  async makeRequest(endpoint, method = 'GET', data = null, headers = {}) {
    const startTime = performance.now();
    const url = this.baseUrl + endpoint;

    try {
      const config = {
        method,
        url,
        headers: {
          'Content-Type': 'application/json',
          ...headers
        },
        timeout: 10000
      };

      if (data) {
        config.data = data;
      }

      const response = await axios(config);
      const endTime = performance.now();
      const responseTime = endTime - startTime;

      this.results.totalRequests++;
      this.results.successfulRequests++;
      this.results.responseTimes.push(responseTime);

      return {
        success: true,
        status: response.status,
        responseTime,
        data: response.data
      };
    } catch (error) {
      const endTime = performance.now();
      const responseTime = endTime - startTime;

      this.results.totalRequests++;
      this.results.failedRequests++;
      this.results.responseTimes.push(responseTime);
      this.results.errors.push({
        endpoint,
        error: error.message,
        responseTime
      });

      return {
        success: false,
        error: error.message,
        responseTime
      };
    }
  }

  async runLoadTest(endpoint, concurrentUsers, totalRequests) {
    console.log(`\n🚀 Starting load test: ${endpoint}`);
    console.log(`Concurrent users: ${concurrentUsers}, Total requests: ${totalRequests}`);

    const promises = [];
    const startTime = performance.now();

    // Create concurrent requests
    for (let i = 0; i < totalRequests; i++) {
      promises.push(this.makeRequest(endpoint));

      // Control concurrency
      if (promises.length >= concurrentUsers) {
        await Promise.all(promises.splice(0, concurrentUsers));
      }
    }

    // Wait for remaining requests
    await Promise.all(promises);

    const endTime = performance.now();
    const totalTime = endTime - startTime;

    return this.generateReport(endpoint, totalTime);
  }

  generateReport(endpoint, totalTime) {
    const responseTimes = this.results.responseTimes;
    const sortedTimes = [...responseTimes].sort((a, b) => a - b);

    const report = {
      endpoint,
      totalTime,
      totalRequests: this.results.totalRequests,
      successfulRequests: this.results.successfulRequests,
      failedRequests: this.results.failedRequests,
      successRate: (this.results.successfulRequests / this.results.totalRequests * 100).toFixed(2) + '%',
      requestsPerSecond: (this.results.totalRequests / (totalTime / 1000)).toFixed(2),
      statistics: {
        min: sortedTimes[0]?.toFixed(2) + 'ms',
        max: sortedTimes[sortedTimes.length - 1]?.toFixed(2) + 'ms',
        median: sortedTimes[Math.floor(sortedTimes.length / 2)]?.toFixed(2) + 'ms',
        p95: sortedTimes[Math.floor(sortedTimes.length * 0.95)]?.toFixed(2) + 'ms',
        p99: sortedTimes[Math.floor(sortedTimes.length * 0.99)]?.toFixed(2) + 'ms',
        average: (responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length).toFixed(2) + 'ms'
      },
      errors: this.results.errors.slice(0, 10) // Show first 10 errors
    };

    return report;
  }

  resetResults() {
    this.results = {
      totalRequests: 0,
      successfulRequests: 0,
      failedRequests: 0,
      responseTimes: [],
      errors: []
    };
  }

  printReport(report) {
    console.log('\n📊 Load Test Results:');
    console.log('='.repeat(50));
    console.log(`Endpoint: ${report.endpoint}`);
    console.log(`Duration: ${(report.totalTime / 1000).toFixed(2)}s`);
    console.log(`Total Requests: ${report.totalRequests}`);
    console.log(`Successful: ${report.successfulRequests}`);
    console.log(`Failed: ${report.failedRequests}`);
    console.log(`Success Rate: ${report.successRate}`);
    console.log(`Requests/sec: ${report.requestsPerSecond}`);
    console.log('\nResponse Time Statistics:');
    console.log(`  Min: ${report.statistics.min}`);
    console.log(`  Max: ${report.statistics.max}`);
    console.log(`  Median: ${report.statistics.median}`);
    console.log(`  95th percentile: ${report.statistics.p95}`);
    console.log(`  99th percentile: ${report.statistics.p99}`);
    console.log(`  Average: ${report.statistics.average}`);

    if (report.errors.length > 0) {
      console.log('\n❌ Sample Errors:');
      report.errors.slice(0, 5).forEach((error, i) => {
        console.log(`  ${i + 1}. ${error.endpoint}: ${error.error} (${error.responseTime.toFixed(2)}ms)`);
      });
    }
  }
}

// Test scenarios
async function runPerformanceTests() {
  const tester = new LoadTester();

  console.log('🔬 Messaging App Performance Assessment');
  console.log('=====================================');

  // Test 1: Health endpoint - baseline performance
  console.log('\n🏥 Testing Health Endpoint (Baseline)');
  let report = await tester.runLoadTest(ENDPOINTS.health, 10, 100);
  tester.printReport(report);

  // Test 2: Auth request code endpoint
  console.log('\n🔐 Testing Auth Request Code Endpoint');
  tester.resetResults();
  report = await tester.runLoadTest(ENDPOINTS.authRequestCode, 5, 50);
  tester.printReport(report);

  // Test 3: Messages endpoint (unauthorized - should fail)
  console.log('\n💬 Testing Messages Endpoint (Unauthorized)');
  tester.resetResults();
  report = await tester.runLoadTest(ENDPOINTS.messages, 5, 50);
  tester.printReport(report);

  // Test 4: Increasing load on health endpoint
  console.log('\n⚡ Testing Health Endpoint Under Load');
  const loadLevels = [
    { concurrent: 10, requests: 100 },
    { concurrent: 25, requests: 250 },
    { concurrent: 50, requests: 500 },
    { concurrent: 100, requests: 1000 }
  ];

  for (const level of loadLevels) {
    console.log(`\n  Load Level: ${level.concurrent} concurrent, ${level.requests} total`);
    tester.resetResults();
    report = await tester.runLoadTest(ENDPOINTS.health, level.concurrent, level.requests);
    tester.printReport(report);

    // Brief pause between tests
    await new Promise(resolve => setTimeout(resolve, 1000));
  }

  console.log('\n✅ Performance assessment completed!');
}

// Export for use in other test files
module.exports = { LoadTester, runPerformanceTests };

// Run tests if this file is executed directly
if (require.main === module) {
  runPerformanceTests().catch(console.error);
}
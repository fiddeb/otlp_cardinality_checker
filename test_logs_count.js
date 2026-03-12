import { check } from 'k6';
import { Httpx } from 'https://jslib.k6.io/httpx/0.0.6/index.js';

const session = new Httpx({
  baseURL: 'http://localhost:4318',
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000,
});

export const options = {
  vus: 1,
  iterations: 1,
};

export default function () {
  // Service A, INFO: 10 logs
  sendLogs('ServiceA', 'INFO', 10);
  
  // Service A, ERROR: 5 logs
  sendLogs('ServiceA', 'ERROR', 5);
  
  // Service B, INFO: 20 logs
  sendLogs('ServiceB', 'INFO', 20);
  
  // Service B, WARN: 3 logs
  sendLogs('ServiceB', 'WARN', 3);
  
  console.log('✓ Sent: ServiceA/INFO=10, ServiceA/ERROR=5, ServiceB/INFO=20, ServiceB/WARN=3');
  console.log('✓ Total: 38 logs');
}

function sendLogs(serviceName, severity, count) {
  const severityNumber = {
    'DEBUG': 5,
    'INFO': 9,
    'WARN': 13,
    'ERROR': 17,
    'FATAL': 21,
  }[severity] || 9;
  
  const logRecords = [];
  for (let i = 0; i < count; i++) {
    logRecords.push({
      timeUnixNano: `${Date.now() * 1000000}`,
      severityNumber: severityNumber,
      severityText: severity,
      body: {
        stringValue: `Test log message ${i+1} for ${serviceName}`
      },
      attributes: [
        { key: 'log.id', value: { stringValue: `${serviceName}-${severity}-${i+1}` } },
      ],
    });
  }
  
  const payload = {
    resourceLogs: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: serviceName } },
        ],
      },
      scopeLogs: [{
        scope: {
          name: 'test-logger',
          version: '1.0.0',
        },
        logRecords: logRecords,
      }],
    }],
  };
  
  const res = session.post('/v1/logs', JSON.stringify(payload));
  check(res, {
    [`${serviceName}/${severity} sent`]: (r) => r.status === 200,
  });
}

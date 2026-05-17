import { useEffect } from 'react';

/**
 * useSSE - A hook to consume Server-Sent Events with auto-reconnect and visibility awareness
 * @param {Function} onEvent - Callback function called with parsed event data
 */
export const useSSE = (onEvent) => {
  useEffect(() => {
    let eventSource;
    let reconnectTimeout;

    const connect = () => {
      // Don't connect if page is not visible
      if (document.visibilityState !== 'visible') return;

      // Close existing if any
      if (eventSource) eventSource.close();

      eventSource = new EventSource('/sse', { withCredentials: true });

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          onEvent(data);
        } catch (err) {
          console.error('Failed to parse SSE event', err);
        }
      };

      eventSource.onerror = (err) => {
        console.error('SSE connection error:', err);
        eventSource.close();
        
        // Simple reconnect with 3 second delay if still visible
        if (document.visibilityState === 'visible') {
          reconnectTimeout = setTimeout(connect, 3000);
        }
      };

      eventSource.onopen = () => {
        // SSE connection established
      };
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        connect();
      } else {
        if (eventSource) {
          eventSource.close();
          eventSource = null;
        }
        if (reconnectTimeout) {
          clearTimeout(reconnectTimeout);
        }
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    connect();

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      if (eventSource) {
        eventSource.close();
      }
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
    };
  }, [onEvent]);
};

package com.dhaba.worker;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

/**
 * Takes jobs from the Redis list "orders:queue" and moves each order
 * new -> cooking -> ready, updating Postgres at every step.
 */
@Component
public class KitchenWorker {

    private static final Logger log = LoggerFactory.getLogger(KitchenWorker.class);
    static final String QUEUE_KEY = "orders:queue";
    static final String HEARTBEAT_KEY = "kitchen:heartbeat";
    static final Path HEARTBEAT_FILE = Path.of("/tmp/kitchen-heartbeat");

    private final StringRedisTemplate redis;
    private final JdbcTemplate jdbc;
    private final long cookMillis;
    private final ObjectMapper mapper = new ObjectMapper();

    public KitchenWorker(StringRedisTemplate redis,
                         JdbcTemplate jdbc,
                         @Value("${app.cook-millis:5000}") long cookMillis) {
        this.redis = redis;
        this.jdbc = jdbc;
        this.cookMillis = cookMillis;
    }

    /** Waits up to 2 seconds for a job, then processes it. */
    @Scheduled(fixedDelay = 100)
    public void processNextJob() {
        String raw;
        try {
            raw = redis.opsForList().rightPop(QUEUE_KEY, Duration.ofSeconds(2));
        } catch (Exception e) {
            log.warn("Redis not reachable: {}", e.getMessage());
            sleepQuietly(2000);
            return;
        }
        if (raw == null) {
            return;
        }
        try {
            JsonNode job = mapper.readTree(raw);
            int orderId = job.get("order_id").asInt();
            log.info("Processing order {}", orderId);
            processOrder(orderId);
        } catch (Exception e) {
            log.error("Failed to process job {}: {}", raw, e.getMessage());
        }
    }

    void processOrder(int orderId) {
        updateStatus(orderId, "cooking");
        sleepQuietly(cookMillis);
        updateStatus(orderId, "ready");
        log.info("Order {} is ready", orderId);
    }

    private void updateStatus(int orderId, String status) {
        int rows = jdbc.update("UPDATE orders SET status = ? WHERE id = ?", status, orderId);
        if (rows == 0) {
            log.warn("Order {} not found when setting status {}", orderId, status);
        }
    }

    /**
     * Heartbeat every 10 seconds: a Redis key for anyone watching, and a file
     * that a Kubernetes exec probe can check (this service has no web port).
     */
    @Scheduled(fixedRate = 10_000)
    public void writeHeartbeat() {
        String now = String.valueOf(System.currentTimeMillis());
        try {
            redis.opsForValue().set(HEARTBEAT_KEY, now, Duration.ofSeconds(30));
        } catch (Exception e) {
            log.warn("Heartbeat to Redis failed: {}", e.getMessage());
        }
        try {
            Files.writeString(HEARTBEAT_FILE, now);
        } catch (Exception e) {
            log.warn("Heartbeat file write failed: {}", e.getMessage());
        }
    }

    private static void sleepQuietly(long millis) {
        if (millis <= 0) {
            return;
        }
        try {
            Thread.sleep(millis);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }
}

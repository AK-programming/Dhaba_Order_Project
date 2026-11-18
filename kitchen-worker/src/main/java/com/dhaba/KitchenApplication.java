package com.dhaba;

import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.CommandLineRunner;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication
@EnableScheduling
public class KitchenApplication {

    public static void main(String[] args) {
        SpringApplication.run(KitchenApplication.class, args);
    }

    /** The worker has no web port, so the version is shown in the log at startup. */
    @Bean
    CommandLineRunner logVersion(@Value("${app.version:dev}") String version) {
        return args -> LoggerFactory.getLogger(KitchenApplication.class)
                .info("kitchen-worker version {}", version);
    }
}

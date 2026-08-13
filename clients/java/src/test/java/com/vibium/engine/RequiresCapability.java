package com.vibium.engine;

import org.junit.jupiter.api.extension.ExtendWith;
import org.junit.jupiter.api.Tag;

import java.lang.annotation.ElementType;
import java.lang.annotation.Inherited;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

@Inherited
@Retention(RetentionPolicy.RUNTIME)
@Target({ElementType.TYPE, ElementType.METHOD})
@ExtendWith(CapabilityCondition.class)
@Tag("cross-engine")
public @interface RequiresCapability {
    String[] value();
}

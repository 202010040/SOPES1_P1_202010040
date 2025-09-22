obj-m += sysinfo_so1_202100265.o
obj-m += continfo_so1_202100265.o

KERNEL_DIR = /lib/modules/$(shell uname -r)/build
PWD = $(shell pwd)

# Suprimir warning sobre diferencias de compilador
ccflags-y := -Wno-declaration-after-statement

all:
	make -C $(KERNEL_DIR) M=$(PWD) modules

clean:
	make -C $(KERNEL_DIR) M=$(PWD) clean

install:
	sudo insmod sysinfo_so1_202100265.ko
	sudo insmod continfo_so1_202100265.ko

uninstall:
	sudo rmmod sysinfo_so1_202100265
	sudo rmmod continfo_so1_202100265

test:
	cat /proc/sysinfo_so1_202100265
	cat /proc/continfo_so1_202100265

.PHONY: all clean install uninstall test
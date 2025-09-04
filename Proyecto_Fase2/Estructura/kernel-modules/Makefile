obj-m += ram_202010040.o
obj-m += cpu_202010040.o
obj-m += procesos_202010040.o

KERNEL_DIR = /lib/modules/$(shell uname -r)/build
PWD = $(shell pwd)

all:
	make -C $(KERNEL_DIR) M=$(PWD) modules

clean:
	make -C $(KERNEL_DIR) M=$(PWD) clean

install:
	sudo insmod ram_202010040.ko
	sudo insmod cpu_202010040.ko
	sudo insmod procesos_202010040.ko

uninstall:
	sudo rmmod ram_202010040
	sudo rmmod cpu_202010040
	sudo rmmod procesos_202010040

test:
	cat /proc/ram_202010040
	cat /proc/cpu_202010040
	cat /proc/procesos_202010040

.PHONY: all clean install uninstall test
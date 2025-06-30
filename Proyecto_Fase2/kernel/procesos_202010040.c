#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/proc_fs.h>
#include <linux/uaccess.h>
#include <linux/seq_file.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>

#define PROC_NAME "procesos_202010040"

MODULE_LICENSE("GPL");
MODULE_AUTHOR("202010040");
MODULE_DESCRIPTION("Modulo de monitoreo de procesos del sistema");
MODULE_VERSION("1.0");

static struct proc_dir_entry *proc_entry;

static void get_process_stats(unsigned long *running, unsigned long *total, 
                             unsigned long *sleeping, unsigned long *zombie, 
                             unsigned long *stopped)
{
    struct task_struct *task;
    unsigned int state;
    
    *running = 0;
    *total = 0;
    *sleeping = 0;
    *zombie = 0;
    *stopped = 0;
    
    rcu_read_lock();
    for_each_process(task) {
        (*total)++;
        
        // Usar __state para kernels más recientes
        state = READ_ONCE(task->__state);
        
        switch (state) {
            case TASK_RUNNING:
                (*running)++;
                break;
            case TASK_INTERRUPTIBLE:
            case TASK_UNINTERRUPTIBLE:
                (*sleeping)++;
                break;
            case TASK_STOPPED:
            case TASK_TRACED:
                (*stopped)++;
                break;
            case EXIT_ZOMBIE:
                (*zombie)++;
                break;
            default:
                // Para otros estados no clasificados, los contamos como durmiendo
                (*sleeping)++;
                break;
        }
    }
    rcu_read_unlock();
}

static int procesos_show(struct seq_file *m, void *v)
{
    unsigned long running = 0, total = 0, sleeping = 0, zombie = 0, stopped = 0;
    
    get_process_stats(&running, &total, &sleeping, &zombie, &stopped);
    
    seq_printf(m, "{\n");
    seq_printf(m, "  \"procesos_corriendo\": %lu,\n", running);
    seq_printf(m, "  \"total_procesos\": %lu,\n", total);
    seq_printf(m, "  \"procesos_durmiendo\": %lu,\n", sleeping);
    seq_printf(m, "  \"procesos_zombie\": %lu,\n", zombie);
    seq_printf(m, "  \"procesos_parados\": %lu\n", stopped);
    seq_printf(m, "}\n");
    
    return 0;
}

static int procesos_open(struct inode *inode, struct file *file)
{
    return single_open(file, procesos_show, NULL);
}

static const struct proc_ops procesos_proc_ops = {
    .proc_open = procesos_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init procesos_202010040_init(void)
{
    proc_entry = proc_create(PROC_NAME, 0444, NULL, &procesos_proc_ops);
    if (!proc_entry) {
        printk(KERN_ERR "No se pudo crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }
    
    printk(KERN_INFO "Modulo de procesos cargado: /proc/%s\n", PROC_NAME);
    return 0;
}

static void __exit procesos_202010040_exit(void)
{
    proc_remove(proc_entry);
    printk(KERN_INFO "Modulo de procesos descargado\n");
}

module_init(procesos_202010040_init);
module_exit(procesos_202010040_exit);
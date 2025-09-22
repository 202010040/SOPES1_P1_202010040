// sysinfo_so1_202100265.c
// Módulo /proc que muestra info del kernel y procesos generales
// Compatible con kernels recientes (usa __state + READ_ONCE)
#include <linux/version.h>
#include <linux/init.h>
#include <linux/module.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>
#include <linux/sched/mm.h>
#include <linux/utsname.h>       // utsname()
#include <linux/timekeeping.h>   // ktime_get_real_ts64
#include <linux/time64.h>        // time64_to_tm
#include <linux/compiler.h>      // READ_ONCE
#include <linux/mm.h>
#include <linux/sysinfo.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("202100265");
MODULE_DESCRIPTION("SO1 - sysinfo a /proc/sysinfo_so1_202100265");
MODULE_VERSION("1.0");

#define PROC_FILENAME "sysinfo_so1_202100265"

/* ---------------------------------------------------------
 * Helpers para información de memoria
 * ---------------------------------------------------------*/

static void get_memory_info(unsigned long *total_kb, unsigned long *free_kb, unsigned long *used_kb)
{
    struct sysinfo si;
    
    si_meminfo(&si);
    
    *total_kb = si.totalram * (PAGE_SIZE / 1024);
    *free_kb = si.freeram * (PAGE_SIZE / 1024);
    *used_kb = *total_kb - *free_kb;
}

/* ---------------------------------------------------------
 * Helpers para procesos
 * ---------------------------------------------------------*/

// Obtiene VSZ en KB (memoria virtual)
static unsigned long get_vsz_kb(struct task_struct *task)
{
    struct mm_struct *mm;
    unsigned long vsz = 0;
    
    if (!task)
        return 0;
        
    mm = get_task_mm(task);
    if (mm) {
        vsz = mm->total_vm * (PAGE_SIZE / 1024);
        mmput(mm);
    }
    
    return vsz;
}

// Obtiene RSS en KB (memoria física)
static unsigned long get_rss_kb(struct task_struct *task)
{
    struct mm_struct *mm;
    unsigned long rss = 0;
    
    if (!task)
        return 0;
        
    mm = get_task_mm(task);
    if (mm) {
        rss = get_mm_rss(mm) * (PAGE_SIZE / 1024);
        mmput(mm);
    }
    
    return rss;
}

// Calcula porcentaje de memoria (aproximado)
static int get_memory_percent(unsigned long rss_kb)
{
    struct sysinfo si;
    unsigned long total_kb;
    
    si_meminfo(&si);
    total_kb = si.totalram * (PAGE_SIZE / 1024);
    
    if (total_kb == 0)
        return 0;
        
    return (int)((rss_kb * 100) / total_kb);
}

// Heurística de "porcentaje" CPU según estado de tarea
static int get_cpu_percent(struct task_struct *task)
{
    unsigned long st;

    if (!task)
        return 0;

    if (task_is_running(task))
        return 3;

    st = READ_ONCE(task->__state);
    switch (st) {
    case TASK_INTERRUPTIBLE:
    case TASK_UNINTERRUPTIBLE:
        return 1;
    default:
        return 0;
    }
}

// Devuelve string legible del estado de la tarea
static const char *get_task_state(struct task_struct *task)
{
    unsigned long st;

    if (!task)
        return "NULL";

    if (task_is_running(task))
        return "RUNNING";

    st = READ_ONCE(task->__state);
    switch (st) {
    case TASK_INTERRUPTIBLE:    return "INTERRUPTIBLE";
    case TASK_UNINTERRUPTIBLE:  return "UNINTERRUPTIBLE";
#ifdef __TASK_STOPPED
    case __TASK_STOPPED:        return "STOPPED";
#endif
#ifdef __TASK_TRACED
    case __TASK_TRACED:         return "TRACED";
#endif
#ifdef TASK_PARKED
    case TASK_PARKED:           return "PARKED";
#endif
#ifdef TASK_DEAD
    case TASK_DEAD:             return "DEAD";
#endif
#ifdef TASK_WAKEKILL
    case TASK_WAKEKILL:         return "WAKEKILL";
#endif
#ifdef TASK_WAKING
    case TASK_WAKING:           return "WAKING";
#endif
#ifdef TASK_NOLOAD
    case TASK_NOLOAD:           return "NOLOAD";
#endif
#ifdef TASK_NEW
    case TASK_NEW:              return "NEW";
#endif
#ifdef TASK_STATE_MAX
    case TASK_STATE_MAX:        return "STATE_MAX";
#endif
    default:                    return "OTHER";
    }
}

// Obtiene cmdline del proceso
static void get_task_cmdline(struct task_struct *task, char *buffer, size_t size)
{
    struct mm_struct *mm;
    
    if (!task || !buffer || size == 0) {
        if (buffer && size > 0)
            buffer[0] = '\0';
        return;
    }
    
    mm = get_task_mm(task);
    if (mm) {
        // En un caso real, aquí leeríamos /proc/[pid]/cmdline
        // Por simplicidad, usamos el comm
        snprintf(buffer, size, "%s", task->comm);
        mmput(mm);
    } else {
        snprintf(buffer, size, "[%s]", task->comm);
    }
}

/* ---------------------------------------------------------
 * /proc printer
 * ---------------------------------------------------------*/

static int sysinfo_show(struct seq_file *m, void *v)
{
    struct timespec64 ts;
    struct tm tm;
    unsigned long total_ram_kb, free_ram_kb, used_ram_kb;

    // Fecha/hora en runtime
    ktime_get_real_ts64(&ts);
    time64_to_tm(ts.tv_sec, 0, &tm);
    
    // Información de memoria del sistema
    get_memory_info(&total_ram_kb, &free_ram_kb, &used_ram_kb);
    
    // Formato JSON para facilitar el parseo en GO
    seq_puts(m, "{\n");
    seq_printf(m, "  \"timestamp\": \"%04ld-%02d-%02d %02d:%02d:%02d\",\n",
               (long)tm.tm_year + 1900, tm.tm_mon + 1, tm.tm_mday,
               tm.tm_hour, tm.tm_min, tm.tm_sec);
    
    seq_printf(m, "  \"system\": {\n");
    seq_printf(m, "    \"kernel\": \"%s\",\n", utsname()->release);
    seq_printf(m, "    \"architecture\": \"%s\",\n", utsname()->machine);
    seq_printf(m, "    \"hostname\": \"%s\"\n", utsname()->nodename);
    seq_printf(m, "  },\n");
    
    seq_printf(m, "  \"memory\": {\n");
    seq_printf(m, "    \"total_kb\": %lu,\n", total_ram_kb);
    seq_printf(m, "    \"free_kb\": %lu,\n", free_ram_kb);
    seq_printf(m, "    \"used_kb\": %lu\n", used_ram_kb);
    seq_printf(m, "  },\n");

    // Resumen de procesos
    {
        struct task_struct *t;
        unsigned long total = 0, running = 0, sleeping = 0, other = 0;

        rcu_read_lock();
        for_each_process(t) {
            total++;
            if (task_is_running(t))
                running++;
            else {
                unsigned long st = READ_ONCE(t->__state);
                if (st == TASK_INTERRUPTIBLE || st == TASK_UNINTERRUPTIBLE)
                    sleeping++;
                else
                    other++;
            }
        }
        rcu_read_unlock();

        seq_printf(m, "  \"process_summary\": {\n");
        seq_printf(m, "    \"total\": %lu,\n", total);
        seq_printf(m, "    \"running\": %lu,\n", running);
        seq_printf(m, "    \"sleeping\": %lu,\n", sleeping);
        seq_printf(m, "    \"other\": %lu\n", other);
        seq_printf(m, "  },\n");
    }
    
    seq_puts(m, "  \"processes\": [\n");

    rcu_read_lock();
    {
        struct task_struct *t;
        bool first = true;
        
        for_each_process(t) {
            unsigned long vsz_kb = get_vsz_kb(t);
            unsigned long rss_kb = get_rss_kb(t);
            int cpu_pct = get_cpu_percent(t);
            int mem_pct = get_memory_percent(rss_kb);
            char cmdline[256];
            
            get_task_cmdline(t, cmdline, sizeof(cmdline));
            
            if (!first)
                seq_puts(m, ",\n");
            first = false;
            
            seq_puts(m, "    {\n");
            seq_printf(m, "      \"pid\": %d,\n", t->pid);
            seq_printf(m, "      \"ppid\": %d,\n", 
                      t->real_parent ? t->real_parent->pid : 0);
            seq_printf(m, "      \"name\": \"%s\",\n", t->comm);
            seq_printf(m, "      \"cmdline\": \"%s\",\n", cmdline);
            seq_printf(m, "      \"vsz_kb\": %lu,\n", vsz_kb);
            seq_printf(m, "      \"rss_kb\": %lu,\n", rss_kb);
            seq_printf(m, "      \"memory_percent\": %d,\n", mem_pct);
            seq_printf(m, "      \"cpu_percent\": %d,\n", cpu_pct);
            seq_printf(m, "      \"state\": \"%s\"\n", get_task_state(t));
            seq_puts(m, "    }");
        }
    }
    rcu_read_unlock();

    seq_puts(m, "\n  ]\n");
    seq_puts(m, "}\n");

    return 0;
}

/* ---------------------------------------------------------
 * /proc plumbing
 * ---------------------------------------------------------*/

static int sysinfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, sysinfo_show, NULL);
}

#if LINUX_VERSION_CODE >= KERNEL_VERSION(5,6,0)
static const struct proc_ops sysinfo_proc_ops = {
    .proc_open    = sysinfo_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};
#else
static const struct file_operations sysinfo_proc_ops = {
    .owner   = THIS_MODULE,
    .open    = sysinfo_open,
    .read    = seq_read,
    .llseek  = seq_lseek,
    .release = single_release,
};
#endif

/* ---------------------------------------------------------
 * Init / Exit
 * ---------------------------------------------------------*/

static int __init sysinfo_init(void)
{
    struct proc_dir_entry *e;

    e = proc_create(PROC_FILENAME, 0444, NULL, &sysinfo_proc_ops);
    if (!e) {
        pr_err("sysinfo_so1_202100265: no se pudo crear /proc/%s\n", PROC_FILENAME);
        return -ENOMEM;
    }
    pr_info("sysinfo_so1_202100265: cargado (/proc/%s)\n", PROC_FILENAME);
    return 0;
}

static void __exit sysinfo_exit(void)
{
    remove_proc_entry(PROC_FILENAME, NULL);
    pr_info("sysinfo_so1_202100265: descargado\n");
}

module_init(sysinfo_init);
module_exit(sysinfo_exit);
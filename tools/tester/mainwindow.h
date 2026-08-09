#ifndef MAINWINDOW_H
#define MAINWINDOW_H

#include <QMainWindow>

QT_BEGIN_NAMESPACE
namespace Ui {
class MainWindow;
}
QT_END_NAMESPACE

class MainWindow : public QMainWindow
{
    Q_OBJECT

public:
    explicit MainWindow(QWidget *parent = nullptr);
    ~MainWindow() override;

private slots:
    void on_pushButtonOpen_clicked();

    void on_plainTextEditSource_textChanged();

private:
    Ui::MainWindow *ui;

    QString stSource_;  // содержимое ST — то, что коллега будет отдавать твоему процессу
    QString cResult_;   // содержимое C — то, что коллега получит на выходе
};
#endif // MAINWINDOW_H
